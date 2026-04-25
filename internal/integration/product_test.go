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

func TestProductCatalogPublic(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("list categories", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/categories")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var categories []model.CategoryResponse
		err = json.NewDecoder(resp.Body).Decode(&categories)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(categories), 1)
	})

	t.Run("get category by slug", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/categories/men")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var category model.CategoryResponse
		err = json.NewDecoder(resp.Body).Decode(&category)
		require.NoError(t, err)
		assert.Equal(t, "men", category.Slug)
		assert.Equal(t, "Men", category.Name)
	})

	t.Run("list products empty", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/products")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var list model.ProductListResponse
		err = json.NewDecoder(resp.Body).Decode(&list)
		require.NoError(t, err)
		assert.Empty(t, list.Data)
		assert.Equal(t, int64(0), list.Pagination.Total)
	})

	t.Run("featured products empty", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/products/featured")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var products []model.ProductResponse
		err = json.NewDecoder(resp.Body).Decode(&products)
		require.NoError(t, err)
		assert.Empty(t, products)
	})
}

func TestProductCatalogAdmin(t *testing.T) {
	srv, tdb, cleanup := setupTestServer(t)
	defer cleanup()

	// Register and login as admin
	adminEmail := "admin-product@example.com"
	adminPass := "password123"

	registerAdmin(t, srv.URL, adminEmail, adminPass, "Admin User")
	// Assign admin role via DB
	_, err := tdb.Pool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id)
		SELECT u.id, r.id FROM users u, roles r WHERE u.email = $1 AND r.name = 'admin'
	`, adminEmail)
	require.NoError(t, err)

	adminToken := loginUser(t, srv.URL, adminEmail, adminPass)

	t.Run("create product", func(t *testing.T) {
		reqBody := model.CreateProductRequest{
			Name:        "Ultraboost Light",
			Slug:        "ultraboost-light",
			Description: "Energy return running shoes",
			BasePrice:   8500.00,
			Status:      "active",
			Gender:      "men",
			Sport:       "running",
			Tags:        []string{"new", "best-seller"},
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/admin/products", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var product model.ProductResponse
		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(t, err)
		assert.Equal(t, "Ultraboost Light", product.Name)
		assert.Equal(t, "ultraboost-light", product.Slug)
		assert.Equal(t, 8500.00, product.BasePrice)
	})

	t.Run("get product by slug", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/products/ultraboost-light")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var product model.ProductResponse
		err = json.NewDecoder(resp.Body).Decode(&product)
		require.NoError(t, err)
		assert.Equal(t, "Ultraboost Light", product.Name)
	})

	t.Run("list products with data", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/products")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var list model.ProductListResponse
		err = json.NewDecoder(resp.Body).Decode(&list)
		require.NoError(t, err)
		assert.Len(t, list.Data, 1)
		assert.Equal(t, int64(1), list.Pagination.Total)
	})

	t.Run("featured products with data", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/products/featured")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var products []model.ProductResponse
		err = json.NewDecoder(resp.Body).Decode(&products)
		require.NoError(t, err)
		assert.Len(t, products, 1)
	})

	_ = tdb // suppress unused warning if not used directly
}

func registerAdmin(t *testing.T, baseURL, email, password, name string) {
	t.Helper()

	reqBody := model.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	}
	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func loginUser(t *testing.T, baseURL, email, password string) string {
	t.Helper()

	reqBody := model.LoginRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var authResp model.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)
	return authResp.AccessToken
}
