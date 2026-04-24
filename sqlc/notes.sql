-- name: CreateNote :one
INSERT INTO notes (
    user_id,
    title,
    content
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetNoteByID :one
SELECT * FROM notes
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: ListNotesByUser :many
SELECT * FROM notes
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAllNotes :many
SELECT * FROM notes
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListDeletedNotes :many
SELECT * FROM notes
WHERE deleted_at IS NOT NULL
ORDER BY deleted_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateNote :one
UPDATE notes
SET
    title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteNote :one
UPDATE notes
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: RestoreNote :one
UPDATE notes
SET deleted_at = NULL
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING *;

-- name: CountNotesByUser :one
SELECT COUNT(*) FROM notes
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: CountAllNotes :one
SELECT COUNT(*) FROM notes
WHERE deleted_at IS NULL;

-- name: CountDeletedNotes :one
SELECT COUNT(*) FROM notes
WHERE deleted_at IS NOT NULL;
