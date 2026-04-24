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
