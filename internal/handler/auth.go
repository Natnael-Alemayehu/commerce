package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"starterkit/internal/httputil"
	"starterkit/internal/logger"
	"starterkit/internal/middleware"
	"starterkit/internal/model"
	"starterkit/internal/service"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	service *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// RegisterRoutes registers auth routes on the given router.
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/auth/register", h.Register)
	r.Post("/api/v1/auth/login", h.Login)
	r.Post("/api/v1/auth/refresh", h.Refresh)
	r.Post("/api/v1/auth/logout", h.Logout)
	r.With(middleware.Auth(h.service.JWTManager())).Post("/api/v1/auth/logout-all", h.LogoutAll)
	r.With(middleware.Auth(h.service.JWTManager())).Get("/api/v1/auth/me", h.Me)
	r.With(middleware.Auth(h.service.JWTManager())).Delete("/api/v1/auth/me", h.DeleteMe)
}

// Register godoc
// @Summary     Register a new user
// @Description Create a new user account with email, password, name and phone
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body model.RegisterRequest true "Registration details"
// @Success     201 {object} model.AuthResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     409 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	tokens, user, err := h.service.Register(r.Context(), req.Email, req.Password, req.Name, req.Phone)
	if err != nil {
		if isDuplicateKeyError(err) {
			httputil.RespondError(w, r, http.StatusConflict, "CONFLICT", "user with this email already exists")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to register user", err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, model.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		User:         toUserResponse(user),
	})
}

// Login godoc
// @Summary     Login user
// @Description Authenticate a user with email and password
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body model.LoginRequest true "Login credentials"
// @Success     200 {object} model.AuthResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	tokens, user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, service.ErrInvalidCredentials) {
			httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to login", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, model.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		User:         toUserResponse(user),
	})
}

// Refresh godoc
// @Summary     Refresh access token
// @Description Get a new access token using a refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body model.RefreshRequest true "Refresh token"
// @Success     200 {object} model.AuthResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	tokens, user, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, model.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		User:         toUserResponse(user),
	})
}

// Logout godoc
// @Summary     Logout user
// @Description Revoke a specific refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body model.LogoutRequest true "Refresh token to revoke"
// @Success     204 "No Content"
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LogoutAll godoc
// @Summary     Logout from all devices
// @Description Revoke all refresh tokens for the authenticated user
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     204 "No Content"
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	if err := h.service.LogoutAll(r.Context(), userID); err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to logout all sessions", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Me godoc
// @Summary     Get current user
// @Description Get the currently authenticated user's profile
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} model.UserResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	logger.FromContext(r.Context()).Info("fetching user profile", slog.String("user_id", userID))

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, toUserResponse(user))
}

// DeleteMe godoc
// @Summary     Delete account
// @Description Delete the currently authenticated user's account
// @Tags        auth
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body model.DeleteAccountRequest true "Password confirmation"
// @Success     204 "No Content"
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/auth/me [delete]
func (h *AuthHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	var req model.DeleteAccountRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	if err := h.service.DeleteAccount(r.Context(), userID, req.Password); err != nil {
		if errors.Is(err, service.ErrInvalidPassword) || errors.Is(err, service.ErrUserNotFound) {
			httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete account", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toUserResponse(user store.User) model.UserResponse {
	phone := ""
	if user.Phone.Valid {
		phone = user.Phone.String
	}

	avatarUrl := ""
	if user.AvatarUrl.Valid {
		avatarUrl = user.AvatarUrl.String
	}

	bio := ""
	if user.Bio.Valid {
		bio = user.Bio.String
	}

	return model.UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		Phone:     phone,
		AvatarUrl: avatarUrl,
		Bio:       bio,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint")
}


