package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"starterkit/internal/auth"
	"starterkit/internal/model"
	"starterkit/internal/server"
	"starterkit/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*httptest.Server, *testutil.TestDB, func()) {
	t.Helper()

	tdb := testutil.NewTestDB(t)

	passwordHasher := auth.NewPasswordHasher(auth.PasswordConfig{
		Time:    1,
		Memory:  65536,
		Threads: 4,
		KeyLen:  32,
		SaltLen: 16,
	})
	jwtManager := auth.NewJWTManager(
		"test-secret-key-that-is-32-bytes!",
		15*time.Minute,
		168*time.Hour,
	)

	cfg := testutil.TestConfig()
	router := server.New(tdb.Store, passwordHasher, jwtManager, cfg)
	srv := httptest.NewServer(router)

	cleanup := func() {
		tdb.Cleanup(t)
		tdb.Close()
		srv.Close()
	}

	return srv, tdb, cleanup
}

func TestAuthRegister(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("register new user", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.RegisterRequest{
			Email:    "test@example.com",
			Password: "securepassword123",
			Name:     "Test User",
			Phone:    "+1234567890",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var authResp model.AuthResponse
		err = json.NewDecoder(resp.Body).Decode(&authResp)
		require.NoError(t, err)

		assert.NotEmpty(t, authResp.AccessToken)
		assert.NotEmpty(t, authResp.RefreshToken)
		assert.Equal(t, int64(900), authResp.ExpiresIn)
		assert.Equal(t, "test@example.com", authResp.User.Email)
		assert.Equal(t, "Test User", authResp.User.Name)
		assert.Equal(t, "+1234567890", authResp.User.Phone)
	})

	t.Run("register duplicate email", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.RegisterRequest{
			Email:    "test@example.com",
			Password: "anotherpassword123",
			Name:     "Another User",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestAuthLogin(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	// Register a user first
	reqBody, _ := json.Marshal(model.RegisterRequest{
		Email:    "login@example.com",
		Password: "securepassword123",
		Name:     "Login User",
	})

	resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)
	resp.Body.Close()

	t.Run("login with valid credentials", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.LoginRequest{
			Email:    "login@example.com",
			Password: "securepassword123",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var authResp model.AuthResponse
		err = json.NewDecoder(resp.Body).Decode(&authResp)
		require.NoError(t, err)

		assert.NotEmpty(t, authResp.AccessToken)
		assert.NotEmpty(t, authResp.RefreshToken)
	})

	t.Run("login with invalid password", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.LoginRequest{
			Email:    "login@example.com",
			Password: "wrongpassword",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("login with non-existent user", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "somepassword",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAuthProtectedRoutes(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	// Register and login to get tokens
	reqBody, _ := json.Marshal(model.RegisterRequest{
		Email:    "protected@example.com",
		Password: "securepassword123",
		Name:     "Protected User",
	})

	resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)

	var authResp model.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)
	resp.Body.Close()

	accessToken := authResp.AccessToken
	refreshToken := authResp.RefreshToken

	t.Run("access protected route without auth", func(t *testing.T) {
		resp, err := client.Get(srv.URL + "/api/v1/auth/me")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("get current user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var userResp model.UserResponse
		err = json.NewDecoder(resp.Body).Decode(&userResp)
		require.NoError(t, err)

		assert.Equal(t, "protected@example.com", userResp.Email)
		assert.Equal(t, "Protected User", userResp.Name)
	})

	t.Run("refresh token", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.RefreshRequest{
			RefreshToken: refreshToken,
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var newAuthResp model.AuthResponse
		err = json.NewDecoder(resp.Body).Decode(&newAuthResp)
		require.NoError(t, err)

		assert.NotEmpty(t, newAuthResp.AccessToken)
		assert.NotEmpty(t, newAuthResp.RefreshToken)
		assert.NotEqual(t, refreshToken, newAuthResp.RefreshToken)
	})

	t.Run("logout with old refresh token succeeds (already revoked)", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.LogoutRequest{
			RefreshToken: refreshToken,
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/logout", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Token was already revoked during refresh, so logout is idempotent
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestAuthValidation(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("register with invalid email", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.RegisterRequest{
			Email:    "not-an-email",
			Password: "securepassword123",
			Name:     "Test",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("register with short password", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.RegisterRequest{
			Email:    "test2@example.com",
			Password: "short",
			Name:     "Test",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("register with missing fields", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"email": "test3@example.com",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAuthLogoutAndDelete(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	// Register and login
	reqBody, _ := json.Marshal(model.RegisterRequest{
		Email:    "logout@example.com",
		Password: "securepassword123",
		Name:     "Logout User",
	})

	resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)

	var authResp model.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)
	resp.Body.Close()

	accessToken := authResp.AccessToken
	refreshToken := authResp.RefreshToken

	t.Run("logout", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.LogoutRequest{
			RefreshToken: refreshToken,
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/logout", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("logout-all", func(t *testing.T) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/auth/logout-all", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("delete account with wrong password", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.DeleteAccountRequest{
			Password: "wrongpassword",
		})

		req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/auth/me", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("delete account", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.DeleteAccountRequest{
			Password: "securepassword123",
		})

		req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/auth/me", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("login after deletion fails", func(t *testing.T) {
		reqBody, _ := json.Marshal(model.LoginRequest{
			Email:    "logout@example.com",
			Password: "securepassword123",
		})

		resp, err := client.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
