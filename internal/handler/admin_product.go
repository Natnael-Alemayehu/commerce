package handler

import (
	"fmt"
	"net/http"
	"time"

	"starterkit/internal/httputil"
	"starterkit/internal/model"
	"starterkit/internal/service"
	"starterkit/internal/storage"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AdminProductHandler handles admin product management HTTP requests.
type AdminProductHandler struct {
	productService *service.ProductService
	minioClient    *storage.MinIOClient
}

// NewAdminProductHandler creates a new AdminProductHandler.
func NewAdminProductHandler(productService *service.ProductService, minioClient *storage.MinIOClient) *AdminProductHandler {
	return &AdminProductHandler{
		productService: productService,
		minioClient:    minioClient,
	}
}

// RegisterRoutes registers admin product routes on the given router.
func (h *AdminProductHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler, enforcer interface{}) {
	// We'll register these in server.go directly with proper middleware
}

// CreateProduct godoc
// @Summary     Create product
// @Description Create a new product (admin only)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body model.CreateProductRequest true "Product details"
// @Success     201 {object} model.ProductResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/products [post]
func (h *AdminProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	if req.Slug == "" {
		req.Slug = service.Slugify(req.Name)
	}

	product, err := h.productService.CreateProduct(r.Context(), req)
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create product", err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, h.productServiceToResponse(product))
}

// UpdateProduct godoc
// @Summary     Update product
// @Description Update an existing product (admin only)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Product ID"
// @Param       request body model.UpdateProductRequest true "Product updates"
// @Success     200 {object} model.ProductResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/products/{id} [put]
func (h *AdminProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid product id")
		return
	}

	var req model.UpdateProductRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	product, err := h.productService.UpdateProduct(r.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "product not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update product", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, h.productServiceToResponse(product))
}

// DeleteProduct godoc
// @Summary     Delete product
// @Description Soft-delete a product (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Product ID"
// @Success     204 "No Content"
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/products/{id} [delete]
func (h *AdminProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid product id")
		return
	}

	if err := h.productService.SoftDeleteProduct(r.Context(), id); err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete product", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateVariant godoc
// @Summary     Create variant
// @Description Add a variant to a product (admin only)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Product ID"
// @Param       request body model.CreateVariantRequest true "Variant details"
// @Success     201 {object} model.ProductVariantResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/products/{id}/variants [post]
func (h *AdminProductHandler) CreateVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid product id")
		return
	}

	var req model.CreateVariantRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	variant, err := h.productService.CreateVariant(r.Context(), productID, req)
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create variant", err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, h.variantServiceToResponse(variant))
}

// UpdateVariant godoc
// @Summary     Update variant
// @Description Update a product variant (admin only)
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Variant ID"
// @Param       request body model.UpdateVariantRequest true "Variant updates"
// @Success     200 {object} model.ProductVariantResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/variants/{id} [put]
func (h *AdminProductHandler) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid variant id")
		return
	}

	var req model.UpdateVariantRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	variant, err := h.productService.UpdateVariant(r.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "variant not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update variant", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, h.variantServiceToResponse(variant))
}

// DeleteVariant godoc
// @Summary     Delete variant
// @Description Delete a product variant (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Variant ID"
// @Success     204 "No Content"
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/variants/{id} [delete]
func (h *AdminProductHandler) DeleteVariant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid variant id")
		return
	}

	if err := h.productService.DeleteVariant(r.Context(), id); err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete variant", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PresignedUploadURL godoc
// @Summary     Get presigned upload URL
// @Description Get a presigned URL for direct image upload to MinIO (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       filename query string true "Desired filename"
// @Param       folder   query string false "Folder prefix (e.g. products)" default(products)
// @Success     200 {object} model.PresignedUploadResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/upload [post]
func (h *AdminProductHandler) PresignedUploadURL(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		httputil.RespondError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "filename is required")
		return
	}

	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "products"
	}

	objectName := fmt.Sprintf("%s/%s-%s", folder, uuid.NewString(), filename)
	expiry := 15 * time.Minute

	uploadURL, err := h.minioClient.PresignedUploadURL(r.Context(), objectName, expiry)
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate upload url", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, model.PresignedUploadResponse{
		UploadURL:  uploadURL.String(),
		PublicURL:  h.minioClient.PublicURL(objectName),
		ObjectName: objectName,
	})
}

func (h *AdminProductHandler) productServiceToResponse(p store.Product) model.ProductResponse {
	var categoryID *string
	if p.CategoryID.Valid {
		s := uuid.UUID(p.CategoryID.Bytes).String()
		categoryID = &s
	}

	var compareAtPrice *float64
	if p.CompareAtPrice.Valid {
		cap := numericToFloat64(p.CompareAtPrice)
		compareAtPrice = &cap
	}

	return model.ProductResponse{
		ID:               p.ID.String(),
		CategoryID:       categoryID,
		Name:             p.Name,
		Slug:             p.Slug,
		Description:      p.Description.String,
		ShortDescription: p.ShortDescription.String,
		BasePrice:        numericToFloat64(p.BasePrice),
		CompareAtPrice:   compareAtPrice,
		Status:           p.Status.String,
		Gender:           p.Gender.String,
		Sport:            p.Sport.String,
		Brand:            p.Brand.String,
		Tags:             p.Tags,
		WeightG:          p.WeightG.Int32,
		MaterialInfo:     p.MaterialInfo.String,
		CareInstructions: p.CareInstructions.String,
		AvgRating:        numericToFloat64(p.AvgRating),
		ReviewCount:      p.ReviewCount.Int32,
		SeoTitle:         p.SeoTitle.String,
		SeoDescription:   p.SeoDescription.String,
		CreatedAt:        p.CreatedAt.Time,
		UpdatedAt:        p.UpdatedAt.Time,
	}
}

func (h *AdminProductHandler) variantServiceToResponse(v store.ProductVariant) model.ProductVariantResponse {
	var priceOverride *float64
	if v.PriceOverride.Valid {
		po := numericToFloat64(v.PriceOverride)
		priceOverride = &po
	}

	return model.ProductVariantResponse{
		ID:            v.ID.String(),
		ProductID:     v.ProductID.String(),
		SKU:           v.Sku,
		VariantName:   v.VariantName.String,
		ColorName:     v.ColorName.String,
		ColorHex:      v.ColorHex.String,
		SizeLabel:     v.SizeLabel.String,
		SizeSystem:    v.SizeSystem.String,
		PriceOverride: priceOverride,
		IsActive:      v.IsActive.Bool,
		CreatedAt:     v.CreatedAt.Time,
	}
}

// numericToFloat64 converts pgtype.Numeric to float64.
func numericToFloat64(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}
