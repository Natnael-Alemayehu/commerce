-- name: CreateInventoryItem :one
INSERT INTO inventory (variant_id, quantity, reserved_quantity, low_stock_threshold)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetInventoryByVariant :one
SELECT * FROM inventory WHERE variant_id = $1;

-- name: ListInventory :many
SELECT i.*, v.sku, v.variant_name, p.name as product_name
FROM inventory i
JOIN product_variants v ON i.variant_id = v.id
JOIN products p ON v.product_id = p.id
ORDER BY i.updated_at DESC
LIMIT $1 OFFSET $2;

-- name: CountInventory :one
SELECT COUNT(*) FROM inventory;

-- name: UpdateInventoryQuantity :one
UPDATE inventory
SET quantity = $2, updated_at = NOW()
WHERE variant_id = $1
RETURNING *;

-- name: UpdateInventoryReserved :one
UPDATE inventory
SET reserved_quantity = $2, updated_at = NOW()
WHERE variant_id = $1
RETURNING *;

-- name: ReserveStock :one
UPDATE inventory
SET reserved_quantity = reserved_quantity + $2, updated_at = NOW()
WHERE variant_id = $1 AND (quantity - reserved_quantity) >= $2
RETURNING *;

-- name: ReleaseStock :one
UPDATE inventory
SET reserved_quantity = GREATEST(reserved_quantity - $2, 0), updated_at = NOW()
WHERE variant_id = $1
RETURNING *;

-- name: ConfirmStockDeduction :one
UPDATE inventory
SET quantity = quantity - $2,
    reserved_quantity = GREATEST(reserved_quantity - $2, 0),
    updated_at = NOW()
WHERE variant_id = $1 AND quantity >= $2
RETURNING *;

-- name: DeleteInventoryItem :exec
DELETE FROM inventory WHERE variant_id = $1;

-- name: CreateStockMovement :one
INSERT INTO stock_movements (variant_id, movement_type, quantity, reason, reference_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListStockMovementsByVariant :many
SELECT * FROM stock_movements WHERE variant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListLowStockItems :many
SELECT i.*, v.sku, v.variant_name, p.name as product_name
FROM inventory i
JOIN product_variants v ON i.variant_id = v.id
JOIN products p ON v.product_id = p.id
WHERE i.quantity <= i.low_stock_threshold
ORDER BY i.quantity ASC;
