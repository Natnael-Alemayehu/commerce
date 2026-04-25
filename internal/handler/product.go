package handler

import (
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

// ProductHandler handles public product catalog HTTP requests.
type ProductHandler struct {
	service *service.ProductService
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// RegisterRoutes registers public product routes on the given router.
func (h *ProductHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/products", h.ListProducts)
	r.Get("/api/v1/products/featured", h.ListFeatured)
	r.Get("/api/v1/products/{slug}", h.GetProduct)
	r.Get("/api/v1/categories", h.ListCategories)
	r.Get("/api/v1/categories/{slug}", h.GetCategory)
}

// ListProducts godoc
// @Summary     List products
// @Description List products with filtering, sorting, and pagination
// @Tags        products
// @Produce     json
// @Param       category_id query string false "Category UUID"
// @Param       gender      query string false "men, women, unisex, kids"
// @Param       sport       query string false "Sport type"
// @Param       price_min   query number false "Minimum price"
// @Param       price_max   query number false "Maximum price"
// @Param       tags        query string false "Comma-separated tags"
// @Param       sort_by     query string false "price_asc, price_desc, newest, rating"
// @Param       limit       query int    false "Page limit" default(20)
// @Param       offset      query int    false "Page offset" default(0)
// @Success     200 {object} model.ProductListResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter := parseProductFilter(r)
	resp, err := h.service.ListProducts(r.Context(), filter)
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list products", err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, resp)
}

// GetProduct godoc
// @Summary     Get product
// @Description Get a single product by slug with variants and images
// @Tags        products
// @Produce     json
// @Param       slug path string true "Product slug"
// @Success     200 {object} model.ProductResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/products/{slug} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	product, err := h.service.GetProductBySlug(r.Context(), slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "product not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get product", err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, product)
}

// ListFeatured godoc
// @Summary     List featured products
// @Description List featured and best-seller products
// @Tags        products
// @Produce     json
// @Param       limit query int false "Limit" default(10)
// @Success     200 {array} model.ProductResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/products/featured [get]
func (h *ProductHandler) ListFeatured(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	products, err := h.service.ListFeaturedProducts(r.Context(), int32(limit))
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list featured products", err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, products)
}

// ListCategories godoc
// @Summary     List categories
// @Description List all active categories
// @Tags        products
// @Produce     json
// @Success     200 {array} model.CategoryResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/categories [get]
func (h *ProductHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.ListCategories(r.Context())
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list categories", err)
		return
	}

	resp := make([]model.CategoryResponse, len(categories))
	for i, c := range categories {
		resp[i] = toCategoryResponse(c)
	}
	httputil.RespondJSON(w, http.StatusOK, resp)
}

// GetCategory godoc
// @Summary     Get category
// @Description Get a category by slug
// @Tags        products
// @Produce     json
// @Param       slug path string true "Category slug"
// @Success     200 {object} model.CategoryResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/categories/{slug} [get]
func (h *ProductHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	category, err := h.service.GetCategoryBySlug(r.Context(), slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "category not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get category", err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, toCategoryResponse(category))
}

func parseProductFilter(r *http.Request) model.ProductListFilter {
	q := r.URL.Query()

	var categoryID *string
	if cid := q.Get("category_id"); cid != "" {
		categoryID = &cid
	}

	priceMin, _ := strconv.ParseFloat(q.Get("price_min"), 64)
	priceMax, _ := strconv.ParseFloat(q.Get("price_max"), 64)

	var tags []string
	if t := q.Get("tags"); t != "" {
		tags = []string{t}
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	return model.ProductListFilter{
		CategoryID: categoryID,
		Gender:     q.Get("gender"),
		Sport:      q.Get("sport"),
		Status:     q.Get("status"),
		PriceMin:   priceMin,
		PriceMax:   priceMax,
		Tags:       tags,
		SortBy:     q.Get("sort_by"),
		Limit:      int64(limit),
		Offset:     int64(offset),
	}
}

func toCategoryResponse(c store.Category) model.CategoryResponse {
	var parentID *string
	if c.ParentID.Valid {
		s := uuid.UUID(c.ParentID.Bytes).String()
		parentID = &s
	}

	return model.CategoryResponse{
		ID:          c.ID.String(),
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description.String,
		ParentID:    parentID,
		SortOrder:   c.SortOrder.Int32,
		ImageURL:    c.ImageUrl.String,
		IsActive:    c.IsActive.Bool,
		CreatedAt:   c.CreatedAt.Time,
		UpdatedAt:   c.UpdatedAt.Time,
	}
}
