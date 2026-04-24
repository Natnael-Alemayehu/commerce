package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"starterkit/internal/logger"
	"starterkit/internal/middleware"
	"starterkit/internal/model"
	"starterkit/internal/service"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
)

// NoteHandler handles note-related HTTP requests.
type NoteHandler struct {
	service *service.NoteService
}

// NewNoteHandler creates a new NoteHandler.
func NewNoteHandler(service *service.NoteService) *NoteHandler {
	return &NoteHandler{service: service}
}

// RegisterRoutes registers note routes on the given router.
func (h *NoteHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/notes", h.Create)
	r.Get("/api/v1/notes", h.List)
	r.Get("/api/v1/notes/{id}", h.Get)
	r.Put("/api/v1/notes/{id}", h.Update)
	r.Delete("/api/v1/notes/{id}", h.Delete)
}

// CreateNoteRequest represents a note creation request.
type CreateNoteRequest struct {
	Title   string `json:"title" validate:"required,max=255"`
	Content string `json:"content" validate:"required"`
}

// UpdateNoteRequest represents a note update request.
type UpdateNoteRequest struct {
	Title   string `json:"title,omitempty" validate:"omitempty,max=255"`
	Content string `json:"content,omitempty" validate:"omitempty"`
}

// ListNotesResponse represents a paginated list of notes.
type ListNotesResponse struct {
	Data []model.NoteResponse  `json:"data"`
	Meta model.PaginationMeta  `json:"meta"`
}

// Create godoc
// @Summary     Create a new note
// @Description Create a new note for the authenticated user
// @Tags        notes
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       request body CreateNoteRequest true "Note details"
// @Success     201 {object} model.NoteResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/notes [post]
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	var req CreateNoteRequest
	if err := decodeAndValidate(r, &req); err != nil {
		LogAndRespondValidationError(w, r, err)
		return
	}

	note, err := h.service.CreateNote(r.Context(), userID, req.Title, req.Content)
	if err != nil {
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create note", err)
		return
	}

	respondJSON(w, http.StatusCreated, toNoteResponse(note))
}

// List godoc
// @Summary     List notes
// @Description List notes for the authenticated user (or all notes for admin)
// @Tags        notes
// @Produce     json
// @Security    BearerAuth
// @Param       limit query int false "Number of items to return" default(20)
// @Param       offset query int false "Offset for pagination" default(0)
// @Success     200 {object} ListNotesResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/notes [get]
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	limit := parseIntParam(r, "limit", 20)
	offset := parseIntParam(r, "offset", 0)

	if limit > 100 {
		limit = 100
	}

	notes, total, err := h.service.ListNotes(r.Context(), userID, int32(limit), int32(offset))
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

// Get godoc
// @Summary     Get a note
// @Description Get a single note by ID
// @Tags        notes
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Note ID"
// @Success     200 {object} model.NoteResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/notes/{id} [get]
func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	logger.FromContext(r.Context()).Info("fetching note", slog.String("note_id", noteID))

	note, err := h.service.GetNote(r.Context(), userID, noteID)
	if err != nil {
		if err.Error() == "note not found" {
			respondError(w, r, http.StatusNotFound, "NOT_FOUND", "note not found")
			return
		}
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get note", err)
		return
	}

	respondJSON(w, http.StatusOK, toNoteResponse(note))
}

// Update godoc
// @Summary     Update a note
// @Description Update an existing note
// @Tags        notes
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Note ID"
// @Param       request body UpdateNoteRequest true "Note updates"
// @Success     200 {object} model.NoteResponse
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/notes/{id} [put]
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	var req UpdateNoteRequest
	if err := decodeAndValidate(r, &req); err != nil {
		LogAndRespondValidationError(w, r, err)
		return
	}

	note, err := h.service.UpdateNote(r.Context(), userID, noteID, req.Title, req.Content)
	if err != nil {
		if err.Error() == "note not found" {
			respondError(w, r, http.StatusNotFound, "NOT_FOUND", "note not found")
			return
		}
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update note", err)
		return
	}

	respondJSON(w, http.StatusOK, toNoteResponse(note))
}

// Delete godoc
// @Summary     Delete a note
// @Description Soft-delete a note
// @Tags        notes
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Note ID"
// @Success     204 "No Content"
// @Failure     401 {object} model.ErrorResponse
// @Failure     404 {object} model.ErrorResponse
// @Failure     500 {object} model.ErrorResponse
// @Router      /api/v1/notes/{id} [delete]
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	if err := h.service.DeleteNote(r.Context(), userID, noteID); err != nil {
		if err.Error() == "note not found" {
			respondError(w, r, http.StatusNotFound, "NOT_FOUND", "note not found")
			return
		}
		LogAndRespondError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete note", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toNoteResponse(note store.Note) model.NoteResponse {
	return model.NoteResponse{
		ID:        note.ID.String(),
		UserID:    note.UserID.String(),
		Title:     note.Title,
		Content:   note.Content,
		CreatedAt: note.CreatedAt.Time,
		UpdatedAt: note.UpdatedAt.Time,
	}
}

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
