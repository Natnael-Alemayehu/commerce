-- name: ListCategories :many
SELECT id, name, slug, description, parent_id, sort_order, image_url, is_active, created_at, updated_at
FROM categories
WHERE is_active = true
ORDER BY sort_order, name;

-- name: GetCategoryBySlug :one
SELECT id, name, slug, description, parent_id, sort_order, image_url, is_active, created_at, updated_at
FROM categories
WHERE slug = $1 AND is_active = true;

-- name: CreateCategory :one
INSERT INTO categories (name, slug, description, parent_id, sort_order, image_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, slug, description, parent_id, sort_order, image_url, is_active, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, slug = $3, description = $4, parent_id = $5, sort_order = $6, image_url = $7, is_active = $8, updated_at = NOW()
WHERE id = $1
RETURNING id, name, slug, description, parent_id, sort_order, image_url, is_active, created_at, updated_at;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-- name: CreateProduct :one
INSERT INTO products (
    category_id, name, slug, description, short_description, base_price, compare_at_price,
    status, gender, sport, brand, tags, weight_g, material_info, care_instructions,
    seo_title, seo_description
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- name: GetProductBySlug :one
SELECT * FROM products WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProducts :many
SELECT * FROM products
WHERE deleted_at IS NULL
  AND (NULLIF(sqlc.arg(category_id), '00000000-0000-0000-0000-000000000000'::uuid) IS NULL OR category_id = sqlc.arg(category_id))
  AND (NULLIF(sqlc.arg(gender), '') IS NULL OR gender = sqlc.arg(gender))
  AND (NULLIF(sqlc.arg(sport), '') IS NULL OR sport = sqlc.arg(sport))
  AND (NULLIF(sqlc.arg(status), '') IS NULL OR status = sqlc.arg(status))
  AND (NULLIF(sqlc.arg(price_min), 0)::decimal IS NULL OR base_price >= sqlc.arg(price_min))
  AND (NULLIF(sqlc.arg(price_max), 0)::decimal IS NULL OR base_price <= sqlc.arg(price_max))
  AND (sqlc.arg(tags)::text[] IS NULL OR tags && sqlc.arg(tags))
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::varchar = 'price_asc' THEN base_price END ASC,
  CASE WHEN sqlc.arg(sort_by)::varchar = 'price_desc' THEN base_price END DESC,
  CASE WHEN sqlc.arg(sort_by)::varchar = 'newest' THEN created_at END DESC,
  CASE WHEN sqlc.arg(sort_by)::varchar = 'rating' THEN avg_rating END DESC,
  CASE WHEN sqlc.arg(sort_by)::varchar = '' OR sqlc.arg(sort_by)::varchar IS NULL THEN created_at END DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProducts :one
SELECT COUNT(*) FROM products
WHERE deleted_at IS NULL
  AND (NULLIF(sqlc.arg(category_id), '00000000-0000-0000-0000-000000000000'::uuid) IS NULL OR category_id = sqlc.arg(category_id))
  AND (NULLIF(sqlc.arg(gender), '') IS NULL OR gender = sqlc.arg(gender))
  AND (NULLIF(sqlc.arg(sport), '') IS NULL OR sport = sqlc.arg(sport))
  AND (NULLIF(sqlc.arg(status), '') IS NULL OR status = sqlc.arg(status))
  AND (NULLIF(sqlc.arg(price_min), 0)::decimal IS NULL OR base_price >= sqlc.arg(price_min))
  AND (NULLIF(sqlc.arg(price_max), 0)::decimal IS NULL OR base_price <= sqlc.arg(price_max))
  AND (sqlc.arg(tags)::text[] IS NULL OR tags && sqlc.arg(tags));

-- name: UpdateProduct :one
UPDATE products
SET category_id = $2, name = $3, slug = $4, description = $5, short_description = $6,
    base_price = $7, compare_at_price = $8, status = $9, gender = $10, sport = $11,
    brand = $12, tags = $13, weight_g = $14, material_info = $15, care_instructions = $16,
    seo_title = $17, seo_description = $18, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products SET deleted_at = NOW(), status = 'discontinued' WHERE id = $1;

-- name: ListFeaturedProducts :many
SELECT * FROM products
WHERE deleted_at IS NULL AND status = 'active'
  AND ('new' = ANY(tags) OR 'best-seller' = ANY(tags))
ORDER BY created_at DESC
LIMIT $1;

-- name: CreateVariant :one
INSERT INTO product_variants (
    product_id, sku, variant_name, color_name, color_hex, size_label, size_system, price_override
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetVariantByID :one
SELECT * FROM product_variants WHERE id = $1;

-- name: ListVariantsByProduct :many
SELECT * FROM product_variants WHERE product_id = $1 AND is_active = true ORDER BY created_at;

-- name: UpdateVariant :one
UPDATE product_variants
SET sku = $2, variant_name = $3, color_name = $4, color_hex = $5,
    size_label = $6, size_system = $7, price_override = $8, is_active = $9
WHERE id = $1
RETURNING *;

-- name: DeleteVariant :exec
DELETE FROM product_variants WHERE id = $1;

-- name: CreateProductImage :one
INSERT INTO product_images (product_id, variant_id, image_url, alt_text, sort_order, is_primary)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListImagesByProduct :many
SELECT * FROM product_images WHERE product_id = $1 ORDER BY sort_order, created_at;

-- name: ListImagesByVariant :many
SELECT * FROM product_images WHERE variant_id = $1 ORDER BY sort_order, created_at;

-- name: SetPrimaryImage :exec
UPDATE product_images SET is_primary = false WHERE product_id = $1;
UPDATE product_images SET is_primary = true WHERE id = $2;

-- name: DeleteProductImage :exec
DELETE FROM product_images WHERE id = $1;

-- name: GetProductImageByID :one
SELECT * FROM product_images WHERE id = $1;
