package service

import (
	"context"
	"errors"
	"fmt"

	"starterkit/internal/rbac"
	"starterkit/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// NoteService handles note business logic.
type NoteService struct {
	store    *store.Store
	enforcer rbac.Enforcer
}

// NewNoteService creates a new NoteService.
func NewNoteService(store *store.Store, enforcer rbac.Enforcer) *NoteService {
	return &NoteService{
		store:    store,
		enforcer: enforcer,
	}
}

// CreateNote creates a new note for a user.
func (s *NoteService) CreateNote(ctx context.Context, userID, title, content string) (store.Note, error) {
	uid := mustParseUUID(userID)

	note, err := s.store.CreateNote(ctx, store.CreateNoteParams{
		UserID:  uid,
		Title:   title,
		Content: content,
	})
	if err != nil {
		return store.Note{}, fmt.Errorf("create note: %w", err)
	}

	return note, nil
}

// GetNote retrieves a note by ID, checking ownership or admin access.
func (s *NoteService) GetNote(ctx context.Context, userID, noteID string) (store.Note, error) {
	uid := mustParseUUID(userID)
	nid := mustParseUUID(noteID)

	note, err := s.store.GetNoteByID(ctx, nid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Note{}, errors.New("note not found")
		}
		return store.Note{}, fmt.Errorf("get note: %w", err)
	}

	// Check ownership or admin permission
	if note.UserID != uid {
		hasPerm, err := s.enforcer.HasPermission(ctx, userID, "notes", "read:all")
		if err != nil {
			return store.Note{}, fmt.Errorf("check permission: %w", err)
		}
		if !hasPerm {
			return store.Note{}, errors.New("note not found")
		}
	}

	return note, nil
}

// ListNotes lists notes for a user or all notes for admin.
func (s *NoteService) ListNotes(ctx context.Context, userID string, limit, offset int32) ([]store.Note, int64, error) {
	hasPerm, err := s.enforcer.HasPermission(ctx, userID, "notes", "list:all")
	if err != nil {
		return nil, 0, fmt.Errorf("check permission: %w", err)
	}

	if hasPerm {
		notes, err := s.store.ListAllNotes(ctx, store.ListAllNotesParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list all notes: %w", err)
		}

		count, err := s.store.CountAllNotes(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("count all notes: %w", err)
		}

		return notes, count, nil
	}

	uid := mustParseUUID(userID)
	notes, err := s.store.ListNotesByUser(ctx, store.ListNotesByUserParams{
		UserID: uid,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}

	count, err := s.store.CountNotesByUser(ctx, uid)
	if err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}

	return notes, count, nil
}

// UpdateNote updates a note, checking ownership or admin access.
func (s *NoteService) UpdateNote(ctx context.Context, userID, noteID, title, content string) (store.Note, error) {
	uid := mustParseUUID(userID)
	nid := mustParseUUID(noteID)

	// Verify ownership or admin permission
	note, err := s.store.GetNoteByID(ctx, nid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Note{}, errors.New("note not found")
		}
		return store.Note{}, fmt.Errorf("get note: %w", err)
	}

	if note.UserID != uid {
		hasPerm, err := s.enforcer.HasPermission(ctx, userID, "notes", "update:all")
		if err != nil {
			return store.Note{}, fmt.Errorf("check permission: %w", err)
		}
		if !hasPerm {
			return store.Note{}, errors.New("note not found")
		}
	}

	titlePg := pgtype.Text{}
	if title != "" {
		titlePg = pgtype.Text{String: title, Valid: true}
	}

	contentPg := pgtype.Text{}
	if content != "" {
		contentPg = pgtype.Text{String: content, Valid: true}
	}

	updated, err := s.store.UpdateNote(ctx, store.UpdateNoteParams{
		ID:      nid,
		Title:   titlePg,
		Content: contentPg,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Note{}, errors.New("note not found")
		}
		return store.Note{}, fmt.Errorf("update note: %w", err)
	}

	return updated, nil
}

// DeleteNote soft-deletes a note, checking ownership or admin access.
func (s *NoteService) DeleteNote(ctx context.Context, userID, noteID string) error {
	uid := mustParseUUID(userID)
	nid := mustParseUUID(noteID)

	// Verify ownership or admin permission
	note, err := s.store.GetNoteByID(ctx, nid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("note not found")
		}
		return fmt.Errorf("get note: %w", err)
	}

	if note.UserID != uid {
		hasPerm, err := s.enforcer.HasPermission(ctx, userID, "notes", "delete:all")
		if err != nil {
			return fmt.Errorf("check permission: %w", err)
		}
		if !hasPerm {
			return errors.New("note not found")
		}
	}

	_, err = s.store.SoftDeleteNote(ctx, nid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("note not found")
		}
		return fmt.Errorf("delete note: %w", err)
	}

	return nil
}

// ListDeletedNotes lists all soft-deleted notes (admin only).
func (s *NoteService) ListDeletedNotes(ctx context.Context, limit, offset int32) ([]store.Note, int64, error) {
	notes, err := s.store.ListDeletedNotes(ctx, store.ListDeletedNotesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list deleted notes: %w", err)
	}

	count, err := s.store.CountDeletedNotes(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count deleted notes: %w", err)
	}

	return notes, count, nil
}

// RestoreNote restores a soft-deleted note (admin only).
func (s *NoteService) RestoreNote(ctx context.Context, noteID string) (store.Note, error) {
	nid := mustParseUUID(noteID)

	note, err := s.store.RestoreNote(ctx, nid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Note{}, errors.New("note not found")
		}
		return store.Note{}, fmt.Errorf("restore note: %w", err)
	}

	return note, nil
}

// ListAllUsers lists all users (admin only).
func (s *NoteService) ListAllUsers(ctx context.Context) ([]store.User, error) {
	users, err := s.store.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
