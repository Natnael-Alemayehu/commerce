package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"starterkit/internal/handler"
	"starterkit/internal/model"
	"starterkit/internal/store"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteCRUD(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	// Register and login
	reqBody, _ := json.Marshal(model.RegisterRequest{
		Email:    "noteuser@example.com",
		Password: "securepassword123",
		Name:     "Note User",
	})

	resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)

	var authResp model.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)
	resp.Body.Close()

	accessToken := authResp.AccessToken

	var createdNote model.NoteResponse

	t.Run("create note", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.CreateNoteRequest{
			Title:   "My First Note",
			Content: "This is the content of my first note.",
		})

		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/notes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&createdNote)
		require.NoError(t, err)

		assert.NotEmpty(t, createdNote.ID)
		assert.Equal(t, "My First Note", createdNote.Title)
		assert.Equal(t, "This is the content of my first note.", createdNote.Content)
	})

	t.Run("get note", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/notes/"+createdNote.ID, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var note model.NoteResponse
		err = json.NewDecoder(resp.Body).Decode(&note)
		require.NoError(t, err)

		assert.Equal(t, createdNote.ID, note.ID)
		assert.Equal(t, createdNote.Title, note.Title)
	})

	t.Run("list notes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/notes", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp handler.ListNotesResponse
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)

		assert.Len(t, listResp.Data, 1)
		assert.Equal(t, int64(1), listResp.Meta.Total)
	})

	t.Run("update note", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.UpdateNoteRequest{
			Title:   "Updated Title",
			Content: "Updated content.",
		})

		req, _ := http.NewRequest("PUT", srv.URL+"/api/v1/notes/"+createdNote.ID, bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var note model.NoteResponse
		err = json.NewDecoder(resp.Body).Decode(&note)
		require.NoError(t, err)

		assert.Equal(t, "Updated Title", note.Title)
		assert.Equal(t, "Updated content.", note.Content)
	})

	t.Run("delete note", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/notes/"+createdNote.ID, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("get deleted note returns 404", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/notes/"+createdNote.ID, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestNoteRBAC(t *testing.T) {
	srv, tdb, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	// Register two users
	user1Req, _ := json.Marshal(model.RegisterRequest{
		Email:    "user1@example.com",
		Password: "securepassword123",
		Name:     "User One",
	})
	resp, err := client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(user1Req))
	require.NoError(t, err)
	var user1Auth model.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&user1Auth)
	require.NoError(t, err)
	resp.Body.Close()

	user2Req, _ := json.Marshal(model.RegisterRequest{
		Email:    "user2@example.com",
		Password: "securepassword123",
		Name:     "User Two",
	})
	resp, err = client.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(user2Req))
	require.NoError(t, err)
	var user2Auth model.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&user2Auth)
	require.NoError(t, err)
	resp.Body.Close()

	// User1 creates a note
	createReq, _ := json.Marshal(handler.CreateNoteRequest{
		Title:   "User1 Note",
		Content: "Content from user1",
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/notes", bytes.NewReader(createReq))
	req.Header.Set("Authorization", "Bearer "+user1Auth.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	var note model.NoteResponse
	err = json.NewDecoder(resp.Body).Decode(&note)
	require.NoError(t, err)
	resp.Body.Close()

	t.Run("user2 cannot access user1 note", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/notes/"+note.ID, nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("user2 cannot update user1 note", func(t *testing.T) {
		updateReq, _ := json.Marshal(handler.UpdateNoteRequest{
			Title: "Hacked",
		})
		req, _ := http.NewRequest("PUT", srv.URL+"/api/v1/notes/"+note.ID, bytes.NewReader(updateReq))
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("user2 cannot delete user1 note", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/notes/"+note.ID, nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("user cannot access admin endpoints", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+user1Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	// Assign admin role to user2 via DB
	ctx := context.Background()
	adminRole, err := tdb.Store.GetRoleByName(ctx, "admin")
	require.NoError(t, err)

	user2ID, err := uuid.Parse(user2Auth.User.ID)
	require.NoError(t, err)

	err = tdb.Store.AssignRoleToUser(ctx, store.AssignRoleToUserParams{
		UserID: user2ID,
		RoleID: adminRole.ID,
	})
	require.NoError(t, err)

	t.Run("admin can access user1 note", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/notes/"+note.ID, nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("admin can list all users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("admin can list all notes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/admin/notes", nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("admin can soft-delete any note", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/notes/"+note.ID, nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("admin can list deleted notes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/admin/notes/deleted", nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("admin can restore deleted note", func(t *testing.T) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/admin/notes/"+note.ID+"/restore", nil)
		req.Header.Set("Authorization", "Bearer "+user2Auth.AccessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var restored model.NoteResponse
		err = json.NewDecoder(resp.Body).Decode(&restored)
		require.NoError(t, err)
		assert.Equal(t, note.ID, restored.ID)
	})
}
