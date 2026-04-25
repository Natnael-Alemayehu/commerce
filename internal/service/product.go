package service

import (
	"context"
	"fmt"
	"strings"

	"starterkit/internal/model"
	"starterkit/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductService handles product catalog business logic.
type ProductService struct {
	store *store.Store
}

// NewProductService creates a new ProductService.
func NewProductService(store *store.Store) *ProductService {
	return &ProductService{store: store}
}

// numericToFloat64 converts pgtype.Numeric to float64.
func numericToFloat64(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

// float64ToNumeric converts float64 to pgtype.Numeric.
func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%f", f))
	return n
}

// Category Methods

func (s *ProductService) ListCategories(ctx context.Context) ([]store.Category, error) {
	return s.store.ListCategories(ctx)
}

func (s *ProductService) GetCategoryBySlug(ctx context.Context, slug string) (store.Category, error) {
	return s.store.GetCategoryBySlug(ctx, slug)
}

func (s *ProductService) CreateCategory(ctx context.Context, req model.CreateProductRequest) error {
	// Not needed for MVP - categories seeded via migration
	return nil
}

// Product Methods

func (s *ProductService) ListProducts(ctx context.Context, filter model.ProductListFilter) (model.ProductListResponse, error) {
	var categoryID uuid.UUID
	if filter.CategoryID != nil && *filter.CategoryID != "" {
		cid, err := uuid.Parse(*filter.CategoryID)
		if err != nil {
			return model.ProductListResponse{}, fmt.Errorf("invalid category_id: %w", err)
		}
		categoryID = cid
	}

	params := store.ListProductsParams{
		CategoryID: categoryID,
		Gender:     filter.Gender,
		Sport:      filter.Sport,
		Status:     filter.Status,
		PriceMin:   float64ToNumeric(filter.PriceMin),
		PriceMax:   float64ToNumeric(filter.PriceMax),
		Tags:       filter.Tags,
		SortBy:     filter.SortBy,
		PageLimit:  int32(filter.Limit),
		PageOffset: int32(filter.Offset),
	}

	products, err := s.store.ListProducts(ctx, params)
	if err != nil {
		return model.ProductListResponse{}, fmt.Errorf("list products: %w", err)
	}

	count, err := s.store.CountProducts(ctx, store.CountProductsParams{
		CategoryID: categoryID,
		Gender:     filter.Gender,
		Sport:      filter.Sport,
		Status:     filter.Status,
		PriceMin:   float64ToNumeric(filter.PriceMin),
		PriceMax:   float64ToNumeric(filter.PriceMax),
		Tags:       filter.Tags,
	})
	if err != nil {
		return model.ProductListResponse{}, fmt.Errorf("count products: %w", err)
	}

	resp := make([]model.ProductResponse, len(products))
	for i, p := range products {
		resp[i] = s.toProductResponse(p)
	}

	return model.ProductListResponse{
		Data: resp,
		Pagination: model.PaginationMeta{
			Total:  count,
			Limit:  int64(filter.Limit),
			Offset: int64(filter.Offset),
		},
	}, nil
}

func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (model.ProductResponse, error) {
	product, err := s.store.GetProductBySlug(ctx, slug)
	if err != nil {
		return model.ProductResponse{}, fmt.Errorf("get product: %w", err)
	}

	resp := s.toProductResponse(product)
	resp.Variants, _ = s.ListVariantsByProduct(ctx, product.ID)
	resp.Images, _ = s.ListImagesByProduct(ctx, product.ID)

	return resp, nil
}

func (s *ProductService) GetProductByID(ctx context.Context, id uuid.UUID) (store.Product, error) {
	return s.store.GetProductByID(ctx, id)
}

func (s *ProductService) CreateProduct(ctx context.Context, req model.CreateProductRequest) (store.Product, error) {
	var categoryID uuid.UUID
	if req.CategoryID != nil {
		categoryID = uuid.MustParse(*req.CategoryID)
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	brand := req.Brand
	if brand == "" {
		brand = "adidas"
	}

	product, err := s.store.CreateProduct(ctx, store.CreateProductParams{
		CategoryID:       pgtype.UUID{Bytes: categoryID, Valid: categoryID != uuid.Nil},
		Name:             req.Name,
		Slug:             req.Slug,
		Description:      pgtype.Text{String: req.Description, Valid: req.Description != ""},
		ShortDescription: pgtype.Text{String: req.ShortDescription, Valid: req.ShortDescription != ""},
		BasePrice:        float64ToNumeric(req.BasePrice),
		CompareAtPrice:   float64ToNumericPtr(req.CompareAtPrice),
		Status:           pgtype.Text{String: status, Valid: true},
		Gender:           pgtype.Text{String: req.Gender, Valid: req.Gender != ""},
		Sport:            pgtype.Text{String: req.Sport, Valid: req.Sport != ""},
		Brand:            pgtype.Text{String: brand, Valid: true},
		Tags:             req.Tags,
		WeightG:          pgtype.Int4{Int32: req.WeightG, Valid: req.WeightG > 0},
		MaterialInfo:     pgtype.Text{String: req.MaterialInfo, Valid: req.MaterialInfo != ""},
		CareInstructions: pgtype.Text{String: req.CareInstructions, Valid: req.CareInstructions != ""},
		SeoTitle:         pgtype.Text{String: req.SeoTitle, Valid: req.SeoTitle != ""},
		SeoDescription:   pgtype.Text{String: req.SeoDescription, Valid: req.SeoDescription != ""},
	})
	if err != nil {
		return store.Product{}, fmt.Errorf("create product: %w", err)
	}
	return product, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, id uuid.UUID, req model.UpdateProductRequest) (store.Product, error) {
	product, err := s.store.GetProductByID(ctx, id)
	if err != nil {
		return store.Product{}, fmt.Errorf("get product: %w", err)
	}

	// Build update params, preserving existing values for omitted fields
	params := store.UpdateProductParams{
		ID: id,
	}

	if req.CategoryID != nil {
		cid := uuid.MustParse(*req.CategoryID)
		params.CategoryID = pgtype.UUID{Bytes: cid, Valid: cid != uuid.Nil}
	} else {
		params.CategoryID = product.CategoryID
	}

	if req.Name != "" {
		params.Name = req.Name
	} else {
		params.Name = product.Name
	}

	if req.Slug != "" {
		params.Slug = req.Slug
	} else {
		params.Slug = product.Slug
	}

	if req.Description != "" {
		params.Description = pgtype.Text{String: req.Description, Valid: true}
	} else {
		params.Description = product.Description
	}

	if req.ShortDescription != "" {
		params.ShortDescription = pgtype.Text{String: req.ShortDescription, Valid: true}
	} else {
		params.ShortDescription = product.ShortDescription
	}

	if req.BasePrice != nil {
		params.BasePrice = float64ToNumeric(*req.BasePrice)
	} else {
		params.BasePrice = product.BasePrice
	}

	if req.CompareAtPrice != nil {
		params.CompareAtPrice = float64ToNumericPtr(req.CompareAtPrice)
	} else {
		params.CompareAtPrice = product.CompareAtPrice
	}

	if req.Status != "" {
		params.Status = pgtype.Text{String: req.Status, Valid: true}
	} else {
		params.Status = product.Status
	}

	if req.Gender != "" {
		params.Gender = pgtype.Text{String: req.Gender, Valid: true}
	} else {
		params.Gender = product.Gender
	}

	if req.Sport != "" {
		params.Sport = pgtype.Text{String: req.Sport, Valid: true}
	} else {
		params.Sport = product.Sport
	}

	if req.Brand != "" {
		params.Brand = pgtype.Text{String: req.Brand, Valid: true}
	} else {
		params.Brand = product.Brand
	}

	if req.Tags != nil {
		params.Tags = req.Tags
	} else {
		params.Tags = product.Tags
	}

	if req.WeightG != nil {
		params.WeightG = pgtype.Int4{Int32: *req.WeightG, Valid: true}
	} else {
		params.WeightG = product.WeightG
	}

	if req.MaterialInfo != "" {
		params.MaterialInfo = pgtype.Text{String: req.MaterialInfo, Valid: true}
	} else {
		params.MaterialInfo = product.MaterialInfo
	}

	if req.CareInstructions != "" {
		params.CareInstructions = pgtype.Text{String: req.CareInstructions, Valid: true}
	} else {
		params.CareInstructions = product.CareInstructions
	}

	if req.SeoTitle != "" {
		params.SeoTitle = pgtype.Text{String: req.SeoTitle, Valid: true}
	} else {
		params.SeoTitle = product.SeoTitle
	}

	if req.SeoDescription != "" {
		params.SeoDescription = pgtype.Text{String: req.SeoDescription, Valid: true}
	} else {
		params.SeoDescription = product.SeoDescription
	}

	updated, err := s.store.UpdateProduct(ctx, params)
	if err != nil {
		return store.Product{}, fmt.Errorf("update product: %w", err)
	}
	return updated, nil
}

func (s *ProductService) SoftDeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.store.SoftDeleteProduct(ctx, id)
}

func (s *ProductService) ListFeaturedProducts(ctx context.Context, limit int32) ([]model.ProductResponse, error) {
	products, err := s.store.ListFeaturedProducts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list featured products: %w", err)
	}

	resp := make([]model.ProductResponse, len(products))
	for i, p := range products {
		resp[i] = s.toProductResponse(p)
	}
	return resp, nil
}

// Variant Methods

func (s *ProductService) CreateVariant(ctx context.Context, productID uuid.UUID, req model.CreateVariantRequest) (store.ProductVariant, error) {
	variant, err := s.store.CreateVariant(ctx, store.CreateVariantParams{
		ProductID:     productID,
		Sku:           req.SKU,
		VariantName:   pgtype.Text{String: req.VariantName, Valid: req.VariantName != ""},
		ColorName:     pgtype.Text{String: req.ColorName, Valid: req.ColorName != ""},
		ColorHex:      pgtype.Text{String: req.ColorHex, Valid: req.ColorHex != ""},
		SizeLabel:     pgtype.Text{String: req.SizeLabel, Valid: req.SizeLabel != ""},
		SizeSystem:    pgtype.Text{String: req.SizeSystem, Valid: req.SizeSystem != ""},
		PriceOverride: float64ToNumericPtr(req.PriceOverride),
	})
	if err != nil {
		return store.ProductVariant{}, fmt.Errorf("create variant: %w", err)
	}
	return variant, nil
}

func (s *ProductService) GetVariantByID(ctx context.Context, id uuid.UUID) (store.ProductVariant, error) {
	return s.store.GetVariantByID(ctx, id)
}

func (s *ProductService) ListVariantsByProduct(ctx context.Context, productID uuid.UUID) ([]model.ProductVariantResponse, error) {
	variants, err := s.store.ListVariantsByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}

	resp := make([]model.ProductVariantResponse, len(variants))
	for i, v := range variants {
		resp[i] = s.toVariantResponse(v)
	}
	return resp, nil
}

func (s *ProductService) UpdateVariant(ctx context.Context, id uuid.UUID, req model.UpdateVariantRequest) (store.ProductVariant, error) {
	variant, err := s.store.GetVariantByID(ctx, id)
	if err != nil {
		return store.ProductVariant{}, fmt.Errorf("get variant: %w", err)
	}

	params := store.UpdateVariantParams{ID: id}

	if req.SKU != "" {
		params.Sku = req.SKU
	} else {
		params.Sku = variant.Sku
	}

	if req.VariantName != "" {
		params.VariantName = pgtype.Text{String: req.VariantName, Valid: true}
	} else {
		params.VariantName = variant.VariantName
	}

	if req.ColorName != "" {
		params.ColorName = pgtype.Text{String: req.ColorName, Valid: true}
	} else {
		params.ColorName = variant.ColorName
	}

	if req.ColorHex != "" {
		params.ColorHex = pgtype.Text{String: req.ColorHex, Valid: true}
	} else {
		params.ColorHex = variant.ColorHex
	}

	if req.SizeLabel != "" {
		params.SizeLabel = pgtype.Text{String: req.SizeLabel, Valid: true}
	} else {
		params.SizeLabel = variant.SizeLabel
	}

	if req.SizeSystem != "" {
		params.SizeSystem = pgtype.Text{String: req.SizeSystem, Valid: true}
	} else {
		params.SizeSystem = variant.SizeSystem
	}

	if req.PriceOverride != nil {
		params.PriceOverride = float64ToNumericPtr(req.PriceOverride)
	} else {
		params.PriceOverride = variant.PriceOverride
	}

	if req.IsActive != nil {
		params.IsActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	} else {
		params.IsActive = variant.IsActive
	}

	updated, err := s.store.UpdateVariant(ctx, params)
	if err != nil {
		return store.ProductVariant{}, fmt.Errorf("update variant: %w", err)
	}
	return updated, nil
}

func (s *ProductService) DeleteVariant(ctx context.Context, id uuid.UUID) error {
	return s.store.DeleteVariant(ctx, id)
}

// Image Methods

func (s *ProductService) CreateProductImage(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, imageURL, altText string, sortOrder int32, isPrimary bool) (store.ProductImage, error) {
	var vid pgtype.UUID
	if variantID != nil {
		vid = pgtype.UUID{Bytes: *variantID, Valid: true}
	}

	if isPrimary {
		_ = s.store.SetPrimaryImage(ctx, productID)
	}

	image, err := s.store.CreateProductImage(ctx, store.CreateProductImageParams{
		ProductID: productID,
		VariantID: vid,
		ImageUrl:  imageURL,
		AltText:   pgtype.Text{String: altText, Valid: altText != ""},
		SortOrder: pgtype.Int4{Int32: sortOrder, Valid: true},
		IsPrimary: pgtype.Bool{Bool: isPrimary, Valid: true},
	})
	if err != nil {
		return store.ProductImage{}, fmt.Errorf("create image: %w", err)
	}
	return image, nil
}

func (s *ProductService) ListImagesByProduct(ctx context.Context, productID uuid.UUID) ([]model.ProductImageResponse, error) {
	images, err := s.store.ListImagesByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}

	resp := make([]model.ProductImageResponse, len(images))
	for i, img := range images {
		resp[i] = s.toImageResponse(img)
	}
	return resp, nil
}

func (s *ProductService) DeleteProductImage(ctx context.Context, id uuid.UUID) error {
	return s.store.DeleteProductImage(ctx, id)
}

// Conversion helpers

func (s *ProductService) toProductResponse(p store.Product) model.ProductResponse {
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

func (s *ProductService) toVariantResponse(v store.ProductVariant) model.ProductVariantResponse {
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

func (s *ProductService) toImageResponse(img store.ProductImage) model.ProductImageResponse {
	var variantID *string
	if img.VariantID.Valid {
		s := uuid.UUID(img.VariantID.Bytes).String()
		variantID = &s
	}

	return model.ProductImageResponse{
		ID:        img.ID.String(),
		ProductID: img.ProductID.String(),
		VariantID: variantID,
		ImageURL:  img.ImageUrl,
		AltText:   img.AltText.String,
		SortOrder: img.SortOrder.Int32,
		IsPrimary: img.IsPrimary.Bool,
	}
}

func float64ToNumericPtr(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	return float64ToNumeric(*f)
}

// ProductListFilter holds filtering parameters for product listing.
type ProductListFilter struct {
	CategoryID *uuid.UUID
	Gender     string
	Sport      string
	Status     string
	PriceMin   float64
	PriceMax   float64
	Tags       []string
	SortBy     string
	Limit      int64
	Offset     int64
}

// Slugify generates a URL-friendly slug from a string.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// ErrProductNotFound is returned when a product is not found.
var ErrProductNotFound = fmt.Errorf("product not found")

// ErrVariantNotFound is returned when a variant is not found.
var ErrVariantNotFound = fmt.Errorf("variant not found")
