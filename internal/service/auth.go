package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"starterkit/internal/auth"
	"starterkit/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuthService handles authentication business logic.
type AuthService struct {
	store          *store.Store
	passwordHasher *auth.PasswordHasher
	jwtManager     *auth.JWTManager
}

// NewAuthService creates a new AuthService.
func NewAuthService(store *store.Store, passwordHasher *auth.PasswordHasher, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		store:          store,
		passwordHasher: passwordHasher,
		jwtManager:     jwtManager,
	}
}

// Register creates a new user and returns tokens.
func (s *AuthService) Register(ctx context.Context, email, password, name, phone string) (*auth.TokenPair, store.User, error) {
	hash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return nil, store.User{}, fmt.Errorf("hash password: %w", err)
	}

	phonePg := pgtype.Text{}
	if phone != "" {
		phonePg = pgtype.Text{String: phone, Valid: true}
	}

	var user store.User
	if err := s.store.ExecTx(ctx, func(q *store.Queries) error {
		var err error
		user, err = q.CreateUser(ctx, store.CreateUserParams{
			Email:        email,
			PasswordHash: hash,
			Name:         name,
			Phone:        phonePg,
		})
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		role, err := q.GetRoleByName(ctx, "user")
		if err != nil {
			return fmt.Errorf("get user role: %w", err)
		}

		if err := q.AssignRoleToUser(ctx, store.AssignRoleToUserParams{
			UserID: user.ID,
			RoleID: role.ID,
		}); err != nil {
			return fmt.Errorf("assign role: %w", err)
		}

		return nil
	}); err != nil {
		return nil, store.User{}, err
	}

	tokens, err := s.generateAndStoreTokenPair(ctx, user.ID)
	if err != nil {
		return nil, store.User{}, fmt.Errorf("generate tokens: %w", err)
	}

	return tokens, user, nil
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*auth.TokenPair, store.User, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.User{}, ErrInvalidCredentials
		}
		return nil, store.User{}, fmt.Errorf("get user: %w", err)
	}

	valid, err := s.passwordHasher.Verify(password, user.PasswordHash)
	if err != nil || !valid {
		return nil, store.User{}, ErrInvalidCredentials
	}

	tokens, err := s.generateAndStoreTokenPair(ctx, user.ID)
	if err != nil {
		return nil, store.User{}, fmt.Errorf("generate tokens: %w", err)
	}

	return tokens, user, nil
}

// Refresh validates a refresh token and issues a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, store.User, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, store.User{}, ErrInvalidToken
	}

	// Check if token hash exists and is not revoked
	tokenHash := hashToken(refreshToken)
	_, err = s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.User{}, ErrTokenNotFound
		}
		return nil, store.User{}, fmt.Errorf("get refresh token: %w", err)
	}

	// Revoke old token
	if err := s.store.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, store.User{}, fmt.Errorf("revoke old token: %w", err)
	}

	userID, err := parseUUID(claims.UserID)
	if err != nil {
		return nil, store.User{}, fmt.Errorf("invalid user id in token: %w", err)
	}

	// Verify user exists
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.User{}, ErrUserNotFound
		}
		return nil, store.User{}, fmt.Errorf("get user: %w", err)
	}

	tokens, err := s.generateAndStoreTokenPair(ctx, userID)
	if err != nil {
		return nil, store.User{}, fmt.Errorf("generate tokens: %w", err)
	}

	return tokens, user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (store.User, error) {
	id, err := parseUUID(userID)
	if err != nil {
		return store.User{}, err
	}
	return s.store.GetUserByID(ctx, id)
}

// Logout revokes a specific refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	// Validate the token first to ensure it's legitimate
	_, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return ErrInvalidToken
	}

	tokenHash := hashToken(refreshToken)
	if err := s.store.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

// LogoutAll revokes all refresh tokens for a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := s.store.RevokeAllUserRefreshTokens(ctx, id); err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}

// DeleteAccount deletes a user account after password verification.
func (s *AuthService) DeleteAccount(ctx context.Context, userID, password string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}

	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get user: %w", err)
	}

	valid, err := s.passwordHasher.Verify(password, user.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidPassword
	}

	if err := s.store.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

// JWTManager returns the JWT manager for middleware use.
func (s *AuthService) JWTManager() *auth.JWTManager {
	return s.jwtManager
}

func (s *AuthService) generateAndStoreTokenPair(ctx context.Context, userID uuid.UUID) (*auth.TokenPair, error) {
	tokens, err := s.jwtManager.GenerateTokenPair(userID.String())
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	// Store refresh token hash
	refreshHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().UTC().Add(s.jwtManager.RefreshTTL())

	_, err = s.store.CreateRefreshToken(ctx, store.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: refreshHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return tokens, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}


