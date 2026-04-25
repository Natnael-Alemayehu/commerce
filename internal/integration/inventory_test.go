package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"starterkit/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryAdmin(t *testing.T) {
	srv, tdb, cleanup := setupTestServer(t)
	defer cleanup()

	// Register and login as admin
	adminEmail := "admin-inventory@example.com"
	adminPass := "password123"

	registerAdmin(t, srv.URL, adminEmail, adminPass, "Admin User")
	_, err := tdb.Pool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id)
		SELECT u.id, r.id FROM users u, roles r WHERE u.email = $1 AND r.name = 'admin'
	`, adminEmail)
	require.NoError(t, err)
	adminToken := loginUser(t, srv.URL, adminEmail, adminPass)

	// Create a product and variant first
	productReq := model.CreateProductRequest{
		Name:      "Test Running Shoe",
		Slug:      "test-running-shoe",
		BasePrice: 5000.00,
		Status:    "active",
	}
	body, _ := json.Marshal(productReq)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/admin/products", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get the product to find its ID
	productResp, err := http.Get(srv.URL + "/api/v1/products/test-running-shoe")
	require.NoError(t, err)
	var product model.ProductResponse
	err = json.NewDecoder(productResp.Body).Decode(&product)
	productResp.Body.Close()
	require.NoError(t, err)

	// Create a variant
	variantReq := model.CreateVariantRequest{
		SKU:         "TRS-BLK-42",
		VariantName: "Core Black / 42",
		ColorName:   "Core Black",
		SizeLabel:   "42",
	}
	body, _ = json.Marshal(variantReq)
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/admin/products/"+product.ID+"/variants", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	var variant model.ProductVariantResponse
	err = json.NewDecoder(resp.Body).Decode(&variant)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	t.Run("list inventory empty", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/inventory", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var list model.InventoryListResponse
		err = json.NewDecoder(resp.Body).Decode(&list)
		require.NoError(t, err)
		assert.Empty(t, list.Data)
	})

	t.Run("get inventory not found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/inventory/"+variant.ID, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("adjust stock creates inventory", func(t *testing.T) {
		adjustReq := model.AdjustStockRequest{
			Quantity: 100,
			Reason:   "Initial stock",
		}
		body, _ := json.Marshal(adjustReq)
		req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/admin/inventory/"+variant.ID, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var inv model.InventoryResponse
		err = json.NewDecoder(resp.Body).Decode(&inv)
		require.NoError(t, err)
		assert.Equal(t, variant.ID, inv.VariantID)
		assert.Equal(t, int32(100), inv.Quantity)
		assert.Equal(t, int32(0), inv.ReservedQuantity)
		assert.Equal(t, int32(100), inv.AvailableQuantity)
	})

	t.Run("get inventory by variant", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/inventory/"+variant.ID, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var inv model.InventoryResponse
		err = json.NewDecoder(resp.Body).Decode(&inv)
		require.NoError(t, err)
		assert.Equal(t, int32(100), inv.Quantity)
	})

	t.Run("list inventory with data", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/inventory", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var list model.InventoryListResponse
		err = json.NewDecoder(resp.Body).Decode(&list)
		require.NoError(t, err)
		assert.Len(t, list.Data, 1)
		assert.Equal(t, int64(1), list.Pagination.Total)
	})

	t.Run("list stock movements", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/inventory/"+variant.ID+"/movements", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var movements []model.StockMovementResponse
		err = json.NewDecoder(resp.Body).Decode(&movements)
		require.NoError(t, err)
		assert.Len(t, movements, 1)
		assert.Equal(t, "in", movements[0].MovementType)
	})

	t.Run("adjust stock update", func(t *testing.T) {
		adjustReq := model.AdjustStockRequest{
			Quantity: 50,
			Reason:   "Stock correction",
		}
		body, _ := json.Marshal(adjustReq)
		req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/admin/inventory/"+variant.ID, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var inv model.InventoryResponse
		err = json.NewDecoder(resp.Body).Decode(&inv)
		require.NoError(t, err)
		assert.Equal(t, int32(50), inv.Quantity)
	})

	t.Run("low stock list", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/admin/inventory/low-stock", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var items []model.InventoryResponse
		err = json.NewDecoder(resp.Body).Decode(&items)
		require.NoError(t, err)
		// 50 quantity with threshold 5, so not low stock
		assert.Empty(t, items)
	})
}
