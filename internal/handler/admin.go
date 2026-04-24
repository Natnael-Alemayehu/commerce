package handler

import (
	"net/http"

	"starterkit/internal/model"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
)

// AdminHandler handles admin-only HTTP requests.
type AdminHandler struct {
	store *store.Store
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(store *store.Store) *AdminHandler {
	return &AdminHandler{store: store}
}

// RegisterRoutes registers admin routes on the given router.
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/admin/users", h.ListUsers)
}

// ListUsers godoc
// @Summary     List all users
// @Description List all registered users (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} model.UserResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/users [get]
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.GetAllUsers(r.Context())
	if err != nil {
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users", err)
		return
	}

	resp := make([]model.UserResponse, len(users))
	for i, user := range users {
		resp[i] = toUserResponse(user)
	}

	respondJSON(w, http.StatusOK, resp)
}
