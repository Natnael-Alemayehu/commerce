package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHasher(t *testing.T) {
	cfg := DefaultPasswordConfig()
	hasher := NewPasswordHasher(cfg)

	t.Run("hash and verify password", func(t *testing.T) {
		password := "my-secret-password"

		hash, err := hasher.Hash(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Contains(t, hash, "$argon2id$")

		valid, err := hasher.Verify(password, hash)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("verify wrong password", func(t *testing.T) {
		password := "my-secret-password"
		wrongPassword := "wrong-password"

		hash, err := hasher.Hash(password)
		require.NoError(t, err)

		valid, err := hasher.Verify(wrongPassword, hash)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("verify invalid hash format", func(t *testing.T) {
		valid, err := hasher.Verify("password", "invalid-hash")
		require.Error(t, err)
		assert.False(t, valid)
	})
}

func TestJWTManager(t *testing.T) {
	secret := "test-secret-key-that-is-32-bytes!"
	manager := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)

	t.Run("generate and validate token pair", func(t *testing.T) {
		userID := uuid.NewString()

		pair, err := manager.GenerateTokenPair(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEmpty(t, pair.RefreshToken)
		assert.Equal(t, int64(15*60), pair.ExpiresIn)

		// Validate access token
		claims, err := manager.ValidateAccessToken(pair.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, AccessToken, claims.TokenType)

		// Validate refresh token
		refreshClaims, err := manager.ValidateRefreshToken(pair.RefreshToken)
		require.NoError(t, err)
		assert.Equal(t, userID, refreshClaims.UserID)
		assert.Equal(t, RefreshToken, refreshClaims.TokenType)
	})

	t.Run("validate invalid token", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("wrong token type", func(t *testing.T) {
		userID := uuid.NewString()
		pair, err := manager.GenerateTokenPair(userID)
		require.NoError(t, err)

		// Try to use refresh token as access token
		_, err = manager.ValidateAccessToken(pair.RefreshToken)
		assert.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		shortManager := NewJWTManager(secret, -1*time.Hour, -1*time.Hour)
		userID := uuid.NewString()

		pair, err := shortManager.GenerateTokenPair(userID)
		require.NoError(t, err)

		_, err = shortManager.ValidateAccessToken(pair.AccessToken)
		assert.Error(t, err)
	})

	t.Run("missing user_id", func(t *testing.T) {
		// Create token with empty user_id manually
		managerWithEmptyUser := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)
		_, err := managerWithEmptyUser.GenerateTokenPair("")
		require.NoError(t, err)
	})
}
