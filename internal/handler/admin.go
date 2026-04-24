package handler

import (
	"log/slog"
	"net/http"

	"starterkit/internal/logger"
	"starterkit/internal/middleware"
	"starterkit/internal/model"
	"starterkit/internal/service"

	"github.com/go-chi/chi/v5"
)

// AdminHandler handles admin-only HTTP requests.
type AdminHandler struct {
	noteService *service.NoteService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(noteService *service.NoteService) *AdminHandler {
	return &AdminHandler{noteService: noteService}
}

// RegisterRoutes registers admin routes on the given router.
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/admin/users", h.ListUsers)
	r.Get("/api/v1/admin/notes", h.ListAllNotes)
	r.Get("/api/v1/admin/notes/deleted", h.ListDeletedNotes)
	r.Post("/api/v1/admin/notes/{id}/restore", h.RestoreNote)
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
	users, err := h.noteService.ListAllUsers(r.Context())
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

// ListAllNotes godoc
// @Summary     List all notes
// @Description List all notes across all users (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       limit query int false "Number of items to return" default(20)
// @Param       offset query int false "Offset for pagination" default(0)
// @Success     200 {object} ListNotesResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/notes [get]
func (h *AdminHandler) ListAllNotes(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 20)
	offset := parseIntParam(r, "offset", 0)

	if limit > 100 {
		limit = 100
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	logger.FromContext(r.Context()).Info("admin listing all notes")

	notes, total, err := h.noteService.ListNotes(r.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notes", err)
		return
	}

	resp := ListNotesResponse{
		Data: make([]model.NoteResponse, len(notes)),
		Meta: model.PaginationMeta{
			Total:  total,
			Limit:  int64(limit),
			Offset: int64(offset),
		},
	}
	for i, note := range notes {
		resp.Data[i] = toNoteResponse(note)
	}

	respondJSON(w, http.StatusOK, resp)
}

// ListDeletedNotes godoc
// @Summary     List deleted notes
// @Description List all soft-deleted notes (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       limit query int false "Number of items to return" default(20)
// @Param       offset query int false "Offset for pagination" default(0)
// @Success     200 {object} ListNotesResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/notes/deleted [get]
func (h *AdminHandler) ListDeletedNotes(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 20)
	offset := parseIntParam(r, "offset", 0)

	if limit > 100 {
		limit = 100
	}

	logger.FromContext(r.Context()).Info("admin listing deleted notes")

	notes, total, err := h.noteService.ListDeletedNotes(r.Context(), int32(limit), int32(offset))
	if err != nil {
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deleted notes", err)
		return
	}

	resp := ListNotesResponse{
		Data: make([]model.NoteResponse, len(notes)),
		Meta: model.PaginationMeta{
			Total:  total,
			Limit:  int64(limit),
			Offset: int64(offset),
		},
	}
	for i, note := range notes {
		resp.Data[i] = toNoteResponse(note)
	}

	respondJSON(w, http.StatusOK, resp)
}

// RestoreNote godoc
// @Summary     Restore a deleted note
// @Description Restore a soft-deleted note (admin only)
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Note ID"
// @Success     200 {object} model.NoteResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     403 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/admin/notes/{id}/restore [post]
func (h *AdminHandler) RestoreNote(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "id")

	logger.FromContext(r.Context()).Info("admin restoring note", slog.String("note_id", noteID))

	note, err := h.noteService.RestoreNote(r.Context(), noteID)
	if err != nil {
		if err.Error() == "note not found" {
			respondError(w, r, http.StatusNotFound, "NOT_FOUND", "note not found")
			return
		}
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to restore note", err)
		return
	}

	respondJSON(w, http.StatusOK, toNoteResponse(note))
}
