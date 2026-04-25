package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Sentinel errors for the service layer.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenNotFound      = errors.New("token not found")
)

// parseUUID safely parses a UUID string, returning a wrapped error on failure.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}
