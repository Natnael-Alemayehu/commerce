-- +goose Up
-- Remove note-related permissions and role assignments
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'notes'
);

DELETE FROM permissions WHERE resource = 'notes';

-- Add commerce permissions
INSERT INTO permissions (name, resource, action) VALUES
    -- Product permissions
    ('products:create', 'products', 'create'),
    ('products:update', 'products', 'update'),
    ('products:delete', 'products', 'delete'),
    -- Inventory permissions
    ('inventory:update', 'inventory', 'update'),
    ('inventory:read', 'inventory', 'read'),
    -- Order permissions
    ('orders:list', 'orders', 'list'),
    ('orders:read', 'orders', 'read'),
    ('orders:update', 'orders', 'update'),
    -- Review permissions
    ('reviews:moderate', 'reviews', 'moderate'),
    -- Upload permissions
    ('upload:create', 'upload', 'create')
ON CONFLICT (name) DO NOTHING;

-- Re-assign all permissions to admin role (including new ones)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Update user role to remove old note permissions and keep user-level ones
-- The user role already has the basic user permissions, we'll add commerce
-- user-level permissions as we build the features

-- +goose Down
-- Re-add note permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('notes:create', 'notes', 'create'),
    ('notes:list', 'notes', 'list'),
    ('notes:list:all', 'notes', 'list:all'),
    ('notes:list:deleted', 'notes', 'list:deleted'),
    ('notes:read', 'notes', 'read'),
    ('notes:read:all', 'notes', 'read:all'),
    ('notes:update', 'notes', 'update'),
    ('notes:update:all', 'notes', 'update:all'),
    ('notes:delete', 'notes', 'delete'),
    ('notes:delete:all', 'notes', 'delete:all'),
    ('notes:restore', 'notes', 'restore')
ON CONFLICT (name) DO NOTHING;

-- Remove commerce permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource IN ('products', 'inventory', 'orders', 'reviews', 'upload')
);

DELETE FROM permissions WHERE resource IN ('products', 'inventory', 'orders', 'reviews', 'upload');
