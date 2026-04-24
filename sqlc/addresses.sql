-- name: CreateAddress :one
INSERT INTO addresses (
    user_id,
    label,
    recipient_name,
    phone,
    street_address,
    city,
    state,
    postal_code,
    country,
    is_default
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetAddressByID :one
SELECT * FROM addresses
WHERE id = $1 LIMIT 1;

-- name: ListAddressesByUser :many
SELECT * FROM addresses
WHERE user_id = $1
ORDER BY is_default DESC, created_at DESC;

-- name: UpdateAddress :one
UPDATE addresses
SET
    label = COALESCE(sqlc.narg('label'), label),
    recipient_name = COALESCE(sqlc.narg('recipient_name'), recipient_name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    street_address = COALESCE(sqlc.narg('street_address'), street_address),
    city = COALESCE(sqlc.narg('city'), city),
    state = COALESCE(sqlc.narg('state'), state),
    postal_code = COALESCE(sqlc.narg('postal_code'), postal_code),
    country = COALESCE(sqlc.narg('country'), country),
    is_default = COALESCE(sqlc.narg('is_default'), is_default)
WHERE id = $1
RETURNING *;

-- name: DeleteAddress :exec
DELETE FROM addresses
WHERE id = $1;

-- name: ClearUserDefaultAddresses :exec
UPDATE addresses
SET is_default = FALSE
WHERE user_id = $1;
