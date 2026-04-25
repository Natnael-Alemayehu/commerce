package model

import "time"

// ErrorResponse represents a standardized API error.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Name     string `json:"name" validate:"required,max=255"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,max=50"`
}

// LoginRequest represents a user login request.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents a logout request.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// DeleteAccountRequest represents an account deletion request.
type DeleteAccountRequest struct {
	Password string `json:"password" validate:"required"`
}

// UpdateProfileRequest represents a profile update request.
type UpdateProfileRequest struct {
	Name      string `json:"name,omitempty" validate:"omitempty,max=255"`
	Phone     string `json:"phone,omitempty" validate:"omitempty,max=50"`
	AvatarUrl string `json:"avatar_url,omitempty" validate:"omitempty,url"`
	Bio       string `json:"bio,omitempty" validate:"omitempty,max=1000"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	AvatarUrl string    `json:"avatar_url,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthResponse represents the authentication response with tokens.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// AddressResponse represents an address in API responses.
type AddressResponse struct {
	ID            string    `json:"id"`
	Label         string    `json:"label,omitempty"`
	RecipientName string    `json:"recipient_name"`
	Phone         string    `json:"phone,omitempty"`
	StreetAddress string    `json:"street_address"`
	City          string    `json:"city"`
	State         string    `json:"state,omitempty"`
	PostalCode    string    `json:"postal_code"`
	Country       string    `json:"country"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateAddressRequest represents an address creation request.
type CreateAddressRequest struct {
	Label         string `json:"label,omitempty" validate:"omitempty,max=50"`
	RecipientName string `json:"recipient_name" validate:"required,max=255"`
	Phone         string `json:"phone,omitempty" validate:"omitempty,max=50"`
	StreetAddress string `json:"street_address" validate:"required"`
	City          string `json:"city" validate:"required,max=100"`
	State         string `json:"state,omitempty" validate:"omitempty,max=100"`
	PostalCode    string `json:"postal_code" validate:"required,max=20"`
	Country       string `json:"country,omitempty" validate:"omitempty,max=100"`
	IsDefault     bool   `json:"is_default"`
}

// UpdateAddressRequest represents an address update request.
type UpdateAddressRequest struct {
	Label         string `json:"label,omitempty" validate:"omitempty,max=50"`
	RecipientName string `json:"recipient_name,omitempty" validate:"omitempty,max=255"`
	Phone         string `json:"phone,omitempty" validate:"omitempty,max=50"`
	StreetAddress string `json:"street_address,omitempty" validate:"omitempty"`
	City          string `json:"city,omitempty" validate:"omitempty,max=100"`
	State         string `json:"state,omitempty" validate:"omitempty,max=100"`
	PostalCode    string `json:"postal_code,omitempty" validate:"omitempty,max=20"`
	Country       string `json:"country,omitempty" validate:"omitempty,max=100"`
	IsDefault     *bool  `json:"is_default,omitempty"`
}

// PaginationMeta represents pagination metadata.
type PaginationMeta struct {
	Total  int64 `json:"total"`
	Limit  int64 `json:"limit"`
	Offset int64 `json:"offset"`
}

// ProductResponse represents a product in API responses.
type ProductResponse struct {
	ID               string           `json:"id"`
	CategoryID       *string          `json:"category_id,omitempty"`
	Name             string           `json:"name"`
	Slug             string           `json:"slug"`
	Description      string           `json:"description,omitempty"`
	ShortDescription string           `json:"short_description,omitempty"`
	BasePrice        float64          `json:"base_price"`
	CompareAtPrice   *float64         `json:"compare_at_price,omitempty"`
	Status           string           `json:"status"`
	Gender           string           `json:"gender,omitempty"`
	Sport            string           `json:"sport,omitempty"`
	Brand            string           `json:"brand"`
	Tags             []string         `json:"tags,omitempty"`
	WeightG          int32            `json:"weight_g,omitempty"`
	MaterialInfo     string           `json:"material_info,omitempty"`
	CareInstructions string           `json:"care_instructions,omitempty"`
	AvgRating        float64          `json:"avg_rating"`
	ReviewCount      int32            `json:"review_count"`
	SeoTitle         string           `json:"seo_title,omitempty"`
	SeoDescription   string           `json:"seo_description,omitempty"`
	Images           []ProductImageResponse `json:"images,omitempty"`
	Variants         []ProductVariantResponse `json:"variants,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// ProductVariantResponse represents a product variant in API responses.
type ProductVariantResponse struct {
	ID            string   `json:"id"`
	ProductID     string   `json:"product_id"`
	SKU           string   `json:"sku"`
	VariantName   string   `json:"variant_name,omitempty"`
	ColorName     string   `json:"color_name,omitempty"`
	ColorHex      string   `json:"color_hex,omitempty"`
	SizeLabel     string   `json:"size_label,omitempty"`
	SizeSystem    string   `json:"size_system,omitempty"`
	PriceOverride *float64 `json:"price_override,omitempty"`
	IsActive      bool     `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

// ProductImageResponse represents a product image in API responses.
type ProductImageResponse struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	ImageURL  string  `json:"image_url"`
	AltText   string  `json:"alt_text,omitempty"`
	SortOrder int32   `json:"sort_order"`
	IsPrimary bool    `json:"is_primary"`
}

// CategoryResponse represents a category in API responses.
type CategoryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	ParentID    *string   `json:"parent_id,omitempty"`
	SortOrder   int32     `json:"sort_order"`
	ImageURL    string    `json:"image_url,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateProductRequest represents a product creation request (admin).
type CreateProductRequest struct {
	CategoryID       *string  `json:"category_id,omitempty" validate:"omitempty,uuid"`
	Name             string   `json:"name" validate:"required,max=255"`
	Slug             string   `json:"slug" validate:"required,max=255"`
	Description      string   `json:"description,omitempty"`
	ShortDescription string   `json:"short_description,omitempty"`
	BasePrice        float64  `json:"base_price" validate:"required,gt=0"`
	CompareAtPrice   *float64 `json:"compare_at_price,omitempty"`
	Status           string   `json:"status,omitempty" validate:"omitempty,oneof=active draft discontinued"`
	Gender           string   `json:"gender,omitempty" validate:"omitempty,oneof=men women unisex kids"`
	Sport            string   `json:"sport,omitempty" validate:"omitempty,max=50"`
	Brand            string   `json:"brand,omitempty" validate:"omitempty,max=50"`
	Tags             []string `json:"tags,omitempty"`
	WeightG          int32    `json:"weight_g,omitempty"`
	MaterialInfo     string   `json:"material_info,omitempty"`
	CareInstructions string   `json:"care_instructions,omitempty"`
	SeoTitle         string   `json:"seo_title,omitempty" validate:"omitempty,max=255"`
	SeoDescription   string   `json:"seo_description,omitempty"`
}

// UpdateProductRequest represents a product update request (admin).
type UpdateProductRequest struct {
	CategoryID       *string  `json:"category_id,omitempty" validate:"omitempty,uuid"`
	Name             string   `json:"name,omitempty" validate:"omitempty,max=255"`
	Slug             string   `json:"slug,omitempty" validate:"omitempty,max=255"`
	Description      string   `json:"description,omitempty"`
	ShortDescription string   `json:"short_description,omitempty"`
	BasePrice        *float64 `json:"base_price,omitempty" validate:"omitempty,gt=0"`
	CompareAtPrice   *float64 `json:"compare_at_price,omitempty"`
	Status           string   `json:"status,omitempty" validate:"omitempty,oneof=active draft discontinued"`
	Gender           string   `json:"gender,omitempty" validate:"omitempty,oneof=men women unisex kids"`
	Sport            string   `json:"sport,omitempty" validate:"omitempty,max=50"`
	Brand            string   `json:"brand,omitempty" validate:"omitempty,max=50"`
	Tags             []string `json:"tags,omitempty"`
	WeightG          *int32   `json:"weight_g,omitempty"`
	MaterialInfo     string   `json:"material_info,omitempty"`
	CareInstructions string   `json:"care_instructions,omitempty"`
	SeoTitle         string   `json:"seo_title,omitempty" validate:"omitempty,max=255"`
	SeoDescription   string   `json:"seo_description,omitempty"`
}

// CreateVariantRequest represents a variant creation request (admin).
type CreateVariantRequest struct {
	SKU           string   `json:"sku" validate:"required,max=100"`
	VariantName   string   `json:"variant_name,omitempty" validate:"omitempty,max=100"`
	ColorName     string   `json:"color_name,omitempty" validate:"omitempty,max=50"`
	ColorHex      string   `json:"color_hex,omitempty" validate:"omitempty,max=7"`
	SizeLabel     string   `json:"size_label,omitempty" validate:"omitempty,max=20"`
	SizeSystem    string   `json:"size_system,omitempty" validate:"omitempty,max=10"`
	PriceOverride *float64 `json:"price_override,omitempty"`
}

// UpdateVariantRequest represents a variant update request (admin).
type UpdateVariantRequest struct {
	SKU           string   `json:"sku,omitempty" validate:"omitempty,max=100"`
	VariantName   string   `json:"variant_name,omitempty" validate:"omitempty,max=100"`
	ColorName     string   `json:"color_name,omitempty" validate:"omitempty,max=50"`
	ColorHex      string   `json:"color_hex,omitempty" validate:"omitempty,max=7"`
	SizeLabel     string   `json:"size_label,omitempty" validate:"omitempty,max=20"`
	SizeSystem    string   `json:"size_system,omitempty" validate:"omitempty,max=10"`
	PriceOverride *float64 `json:"price_override,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
}

// ProductListResponse represents a paginated product list response.
type ProductListResponse struct {
	Data       []ProductResponse `json:"data"`
	Pagination PaginationMeta    `json:"pagination"`
}

// PresignedUploadResponse represents a presigned URL for image upload.
type PresignedUploadResponse struct {
	UploadURL  string `json:"upload_url"`
	PublicURL  string `json:"public_url"`
	ObjectName string `json:"object_name"`
}

// ProductListFilter represents filtering parameters for product listing.
type ProductListFilter struct {
	CategoryID *string
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

// InventoryResponse represents an inventory item in API responses.
type InventoryResponse struct {
	ID                string `json:"id"`
	VariantID         string `json:"variant_id"`
	SKU               string `json:"sku"`
	VariantName       string `json:"variant_name,omitempty"`
	ProductName       string `json:"product_name,omitempty"`
	Quantity          int32  `json:"quantity"`
	ReservedQuantity  int32  `json:"reserved_quantity"`
	AvailableQuantity int32  `json:"available_quantity"`
	LowStockThreshold int32  `json:"low_stock_threshold"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// StockMovementResponse represents a stock movement in API responses.
type StockMovementResponse struct {
	ID           string    `json:"id"`
	VariantID    string    `json:"variant_id"`
	MovementType string    `json:"movement_type"`
	Quantity     int32     `json:"quantity"`
	Reason       string    `json:"reason,omitempty"`
	ReferenceID  *string   `json:"reference_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// AdjustStockRequest represents a stock adjustment request (admin).
type AdjustStockRequest struct {
	Quantity int32  `json:"quantity" validate:"required,gte=0"`
	Reason   string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// InventoryListResponse represents a paginated inventory list response.
type InventoryListResponse struct {
	Data       []InventoryResponse `json:"data"`
	Pagination PaginationMeta      `json:"pagination"`
}
