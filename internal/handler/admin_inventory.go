package handler

import (
	"errors"
	"net/http"
	"strconv"

	"starterkit/internal/httputil"
	"starterkit/internal/model"
	"starterkit/internal/service"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AdminInventoryHandler handles admin inventory management HTTP requests.
type AdminInventoryHandler struct {
	inventoryService *service.InventoryService
	productService   *service.ProductService
}

// NewAdminInventoryHandler creates a new AdminInventoryHandler.
func NewAdminInventoryHandler(inventoryService *service.InventoryService, productService *service.ProductService) *AdminInventoryHandler {
	return &AdminInventoryHandler{
		inventoryService: inventoryService,
		productService:   productService,
	}
}

// ListInventory godoc
// @Summary     List inventory
// @Description List all inventory items with product details
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query int false "Page limit" default(20)
// @Param       offset query int false "Page offset" default(0)
// @Success     200 {object} model.InventoryListResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/inventory [get]
func (h *AdminInventoryHandler) ListInventory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	items, count, err := h.inventoryService.ListInventory(r.Context(), int32(limit), int32(offset))
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list inventory", err)
		return
	}

	resp := make([]model.InventoryResponse, len(items))
	for i, item := range items {
		resp[i] = toInventoryResponse(item)
	}

	httputil.RespondJSON(w, http.StatusOK, model.InventoryListResponse{
		Data: resp,
		Pagination: model.PaginationMeta{
			Total:  count,
			Limit:  int64(limit),
			Offset: int64(offset),
		},
	})
}

// GetInventoryByVariant godoc
// @Summary     Get inventory by variant
// @Description Get inventory details for a specific variant
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       variant_id path string true "Variant ID"
// @Success     200 {object} model.InventoryResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/inventory/{variant_id} [get]
func (h *AdminInventoryHandler) GetInventoryByVariant(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid variant id")
		return
	}

	inv, err := h.inventoryService.GetInventoryItem(r.Context(), variantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "inventory not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get inventory", err)
		return
	}

	// Get variant details for response
	variant, _ := h.productService.GetVariantByID(r.Context(), variantID)
	product, _ := h.productService.GetProductByID(r.Context(), variant.ProductID)

	resp := model.InventoryResponse{
		ID:                inv.ID.String(),
		VariantID:         inv.VariantID.String(),
		SKU:               variant.Sku,
		VariantName:       variant.VariantName.String,
		ProductName:       product.Name,
		Quantity:          inv.Quantity,
		ReservedQuantity:  inv.ReservedQuantity,
		AvailableQuantity: inv.Quantity - inv.ReservedQuantity,
		LowStockThreshold: inv.LowStockThreshold.Int32,
		UpdatedAt:         inv.UpdatedAt.Time,
	}

	httputil.RespondJSON(w, http.StatusOK, resp)
}

// AdjustStock godoc
// @Summary     Adjust stock
// @Description Adjust inventory quantity for a variant (admin only)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       variant_id path string true "Variant ID"
// @Param       request body model.AdjustStockRequest true "Stock adjustment"
// @Success     200 {object} model.InventoryResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/inventory/{variant_id} [put]
func (h *AdminInventoryHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid variant id")
		return
	}

	var req model.AdjustStockRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	if err := h.inventoryService.AdjustStock(r.Context(), variantID, req.Quantity, req.Reason); err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to adjust stock", err)
		return
	}

	// Return updated inventory
	inv, err := h.inventoryService.GetInventoryItem(r.Context(), variantID)
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get updated inventory", err)
		return
	}

	variant, _ := h.productService.GetVariantByID(r.Context(), variantID)
	product, _ := h.productService.GetProductByID(r.Context(), variant.ProductID)

	resp := model.InventoryResponse{
		ID:                inv.ID.String(),
		VariantID:         inv.VariantID.String(),
		SKU:               variant.Sku,
		VariantName:       variant.VariantName.String,
		ProductName:       product.Name,
		Quantity:          inv.Quantity,
		ReservedQuantity:  inv.ReservedQuantity,
		AvailableQuantity: inv.Quantity - inv.ReservedQuantity,
		LowStockThreshold: inv.LowStockThreshold.Int32,
		UpdatedAt:         inv.UpdatedAt.Time,
	}

	httputil.RespondJSON(w, http.StatusOK, resp)
}

// ListLowStock godoc
// @Summary     List low stock items
// @Description List inventory items at or below their low stock threshold
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} model.InventoryResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/inventory/low-stock [get]
func (h *AdminInventoryHandler) ListLowStock(w http.ResponseWriter, r *http.Request) {
	items, err := h.inventoryService.ListLowStockItems(r.Context())
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list low stock", err)
		return
	}

	resp := make([]model.InventoryResponse, len(items))
	for i, item := range items {
		resp[i] = model.InventoryResponse{
			ID:                item.ID.String(),
			VariantID:         item.VariantID.String(),
			SKU:               item.Sku,
			VariantName:       item.VariantName.String,
			ProductName:       item.ProductName,
			Quantity:          item.Quantity,
			ReservedQuantity:  item.ReservedQuantity,
			AvailableQuantity: item.Quantity - item.ReservedQuantity,
			LowStockThreshold: item.LowStockThreshold.Int32,
			UpdatedAt:         item.UpdatedAt.Time,
		}
	}

	httputil.RespondJSON(w, http.StatusOK, resp)
}

// ListStockMovements godoc
// @Summary     List stock movements
// @Description List stock movement history for a variant
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       variant_id path string true "Variant ID"
// @Param       limit      query int false "Limit" default(20)
// @Param       offset     query int false "Offset" default(0)
// @Success     200 {array} model.StockMovementResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/inventory/{variant_id}/movements [get]
func (h *AdminInventoryHandler) ListStockMovements(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(chi.URLParam(r, "variant_id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid variant id")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	movements, err := h.inventoryService.ListStockMovements(r.Context(), variantID, int32(limit), int32(offset))
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list movements", err)
		return
	}

	resp := make([]model.StockMovementResponse, len(movements))
	for i, m := range movements {
		var refID *string
		if m.ReferenceID.Valid {
			s := uuid.UUID(m.ReferenceID.Bytes).String()
			refID = &s
		}

		resp[i] = model.StockMovementResponse{
			ID:           m.ID.String(),
			VariantID:    m.VariantID.String(),
			MovementType: m.MovementType,
			Quantity:     m.Quantity,
			Reason:       m.Reason.String,
			ReferenceID:  refID,
			CreatedAt:    m.CreatedAt.Time,
		}
	}

	httputil.RespondJSON(w, http.StatusOK, resp)
}

func toInventoryResponse(item store.ListInventoryRow) model.InventoryResponse {
	return model.InventoryResponse{
		ID:                item.ID.String(),
		VariantID:         item.VariantID.String(),
		SKU:               item.Sku,
		VariantName:       item.VariantName.String,
		ProductName:       item.ProductName,
		Quantity:          item.Quantity,
		ReservedQuantity:  item.ReservedQuantity,
		AvailableQuantity: item.Quantity - item.ReservedQuantity,
		LowStockThreshold: item.LowStockThreshold.Int32,
		UpdatedAt:         item.UpdatedAt.Time,
	}
}
