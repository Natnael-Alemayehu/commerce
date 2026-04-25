package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"starterkit/internal/httputil"
	"starterkit/internal/logger"
	"starterkit/internal/middleware"
	"starterkit/internal/model"
	"starterkit/internal/service"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserHandler handles user profile and address HTTP requests.
type UserHandler struct {
	authService *service.AuthService
	store       *store.Store
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(authService *service.AuthService, store *store.Store) *UserHandler {
	return &UserHandler{
		authService: authService,
		store:       store,
	}
}

// RegisterRoutes registers user routes on the given router.
func (h *UserHandler) RegisterRoutes(r chi.Router) {
	authMiddleware := middleware.Auth(h.authService.JWTManager())

	r.With(authMiddleware).Put("/api/v1/me", h.UpdateProfile)
	r.With(authMiddleware).Get("/api/v1/me/addresses", h.ListAddresses)
	r.With(authMiddleware).Post("/api/v1/me/addresses", h.CreateAddress)
	r.With(authMiddleware).Put("/api/v1/me/addresses/{id}", h.UpdateAddress)
	r.With(authMiddleware).Delete("/api/v1/me/addresses/{id}", h.DeleteAddress)
	r.With(authMiddleware).Put("/api/v1/me/addresses/{id}/default", h.SetDefaultAddress)
}

// UpdateProfile godoc
// @Summary     Update profile
// @Description Update the authenticated user's profile
// @Tags        user
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body model.UpdateProfileRequest true "Profile updates"
// @Success     200 {object} model.UserResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/me [put]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	var req model.UpdateProfileRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	name := pgtype.Text{}
	if req.Name != "" {
		name = pgtype.Text{String: req.Name, Valid: true}
	}

	phone := pgtype.Text{}
	if req.Phone != "" {
		phone = pgtype.Text{String: req.Phone, Valid: true}
	}

	avatarUrl := pgtype.Text{}
	if req.AvatarUrl != "" {
		avatarUrl = pgtype.Text{String: req.AvatarUrl, Valid: true}
	}

	bio := pgtype.Text{}
	if req.Bio != "" {
		bio = pgtype.Text{String: req.Bio, Valid: true}
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid user id")
		return
	}
	user, err := h.store.UpdateUser(r.Context(), store.UpdateUserParams{
		ID:        id,
		Name:      name,
		Phone:     phone,
		AvatarUrl: avatarUrl,
		Bio:       bio,
	})
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update profile", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, toUserResponse(user))
}

// ListAddresses godoc
// @Summary     List addresses
// @Description List all addresses for the authenticated user
// @Tags        user
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} model.AddressResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/me/addresses [get]
func (h *UserHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	logger.FromContext(r.Context()).Info("listing addresses", slog.String("user_id", userID))

	id, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid user id")
		return
	}
	addresses, err := h.store.ListAddressesByUser(r.Context(), id)
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list addresses", err)
		return
	}

	resp := make([]model.AddressResponse, len(addresses))
	for i, addr := range addresses {
		resp[i] = toAddressResponse(addr)
	}

	httputil.RespondJSON(w, http.StatusOK, resp)
}

// CreateAddress godoc
// @Summary     Create address
// @Description Add a new address for the authenticated user
// @Tags        user
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body model.CreateAddressRequest true "Address details"
// @Success     201 {object} model.AddressResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/me/addresses [post]
func (h *UserHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	var req model.CreateAddressRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid user id")
		return
	}

	// If setting as default, clear other defaults first
	if req.IsDefault {
		if err := h.store.ClearUserDefaultAddresses(r.Context(), id); err != nil {
			httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to clear default addresses", err)
			return
		}
	}

	label := pgtype.Text{}
	if req.Label != "" {
		label = pgtype.Text{String: req.Label, Valid: true}
	}

	phone := pgtype.Text{}
	if req.Phone != "" {
		phone = pgtype.Text{String: req.Phone, Valid: true}
	}

	state := pgtype.Text{}
	if req.State != "" {
		state = pgtype.Text{String: req.State, Valid: true}
	}

	country := req.Country
	if country == "" {
		country = "Ethiopia"
	}

	addr, err := h.store.CreateAddress(r.Context(), store.CreateAddressParams{
		UserID:        id,
		Label:         label,
		RecipientName: req.RecipientName,
		Phone:         phone,
		StreetAddress: req.StreetAddress,
		City:          req.City,
		State:         state,
		PostalCode:    req.PostalCode,
		Country:       country,
		IsDefault:     pgtype.Bool{Bool: req.IsDefault, Valid: true},
	})
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create address", err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, toAddressResponse(addr))
}

// UpdateAddress godoc
// @Summary     Update address
// @Description Update an existing address
// @Tags        user
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Address ID"
// @Param       request body model.UpdateAddressRequest true "Address updates"
// @Success     200 {object} model.AddressResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/me/addresses/{id} [put]
func (h *UserHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	addressID := chi.URLParam(r, "id")

	var req model.UpdateAddressRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.LogAndRespondValidationError(w, r, err)
		return
	}

	// Verify ownership
	uid, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid user id")
		return
	}
	aid, err := uuid.Parse(addressID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid address id")
		return
	}
	addr, err := h.store.GetAddressByID(r.Context(), aid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get address", err)
		return
	}

	if addr.UserID != uid {
		httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
		return
	}

	// If setting as default, clear other defaults first
	if req.IsDefault != nil && *req.IsDefault {
		if err := h.store.ClearUserDefaultAddresses(r.Context(), uid); err != nil {
			httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to clear default addresses", err)
			return
		}
	}

	label := pgtype.Text{}
	if req.Label != "" {
		label = pgtype.Text{String: req.Label, Valid: true}
	}

	recipientName := pgtype.Text{}
	if req.RecipientName != "" {
		recipientName = pgtype.Text{String: req.RecipientName, Valid: true}
	}

	phone := pgtype.Text{}
	if req.Phone != "" {
		phone = pgtype.Text{String: req.Phone, Valid: true}
	}

	streetAddress := pgtype.Text{}
	if req.StreetAddress != "" {
		streetAddress = pgtype.Text{String: req.StreetAddress, Valid: true}
	}

	city := pgtype.Text{}
	if req.City != "" {
		city = pgtype.Text{String: req.City, Valid: true}
	}

	state := pgtype.Text{}
	if req.State != "" {
		state = pgtype.Text{String: req.State, Valid: true}
	}

	postalCode := pgtype.Text{}
	if req.PostalCode != "" {
		postalCode = pgtype.Text{String: req.PostalCode, Valid: true}
	}

	country := pgtype.Text{}
	if req.Country != "" {
		country = pgtype.Text{String: req.Country, Valid: true}
	}

	isDefault := pgtype.Bool{}
	if req.IsDefault != nil {
		isDefault = pgtype.Bool{Bool: *req.IsDefault, Valid: true}
	}

	updated, err := h.store.UpdateAddress(r.Context(), store.UpdateAddressParams{
		ID:            aid,
		Label:         label,
		RecipientName: recipientName,
		Phone:         phone,
		StreetAddress: streetAddress,
		City:          city,
		State:         state,
		PostalCode:    postalCode,
		Country:       country,
		IsDefault:     isDefault,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update address", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, toAddressResponse(updated))
}

// DeleteAddress godoc
// @Summary     Delete address
// @Description Delete an address
// @Tags        user
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Address ID"
// @Success     204 "No Content"
// @Failure     401 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/me/addresses/{id} [delete]
func (h *UserHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	addressID := chi.URLParam(r, "id")

	// Verify ownership
	uid, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid user id")
		return
	}
	aid, err := uuid.Parse(addressID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid address id")
		return
	}
	addr, err := h.store.GetAddressByID(r.Context(), aid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get address", err)
		return
	}

	if addr.UserID != uid {
		httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
		return
	}

	if err := h.store.DeleteAddress(r.Context(), aid); err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete address", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultAddress godoc
// @Summary     Set default address
// @Description Set an address as the default
// @Tags        user
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Address ID"
// @Success     200 {object} model.AddressResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/me/addresses/{id}/default [put]
func (h *UserHandler) SetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	addressID := chi.URLParam(r, "id")

	// Verify ownership
	uid, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid user id")
		return
	}
	aid, err := uuid.Parse(addressID)
	if err != nil {
		httputil.RespondError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid address id")
		return
	}
	addr, err := h.store.GetAddressByID(r.Context(), aid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
			return
		}
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get address", err)
		return
	}

	if addr.UserID != uid {
		httputil.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "address not found")
		return
	}

	if err := h.store.ClearUserDefaultAddresses(r.Context(), uid); err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to clear defaults", err)
		return
	}

	updated, err := h.store.UpdateAddress(r.Context(), store.UpdateAddressParams{
		ID:        aid,
		IsDefault: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		httputil.LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set default", err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, toAddressResponse(updated))
}

func toAddressResponse(addr store.Address) model.AddressResponse {
	label := ""
	if addr.Label.Valid {
		label = addr.Label.String
	}

	phone := ""
	if addr.Phone.Valid {
		phone = addr.Phone.String
	}

	state := ""
	if addr.State.Valid {
		state = addr.State.String
	}

	return model.AddressResponse{
		ID:            addr.ID.String(),
		Label:         label,
		RecipientName: addr.RecipientName,
		Phone:         phone,
		StreetAddress: addr.StreetAddress,
		City:          addr.City,
		State:         state,
		PostalCode:    addr.PostalCode,
		Country:       addr.Country,
		IsDefault:     addr.IsDefault.Bool,
		CreatedAt:     addr.CreatedAt.Time,
	}
}
