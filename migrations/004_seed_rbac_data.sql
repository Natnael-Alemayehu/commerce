-- +goose Up
-- Insert default roles
INSERT INTO roles (name, description) VALUES
    ('admin', 'System administrator with full access'),
    ('user', 'Standard user with limited access');

-- Insert permissions
INSERT INTO permissions (name, resource, action) VALUES
    -- User permissions
    ('users:list', 'users', 'list'),
    ('users:read', 'users', 'read'),
    ('users:update', 'users', 'update'),
    ('users:delete', 'users', 'delete'),
    -- Note permissions
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
    ('notes:restore', 'notes', 'restore');

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin';

-- Assign user-level permissions to user role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'user'
  AND p.name IN (
      'users:read', 'users:update', 'users:delete',
      'notes:create', 'notes:list', 'notes:read', 'notes:update', 'notes:delete'
  );

-- +goose Down
DELETE FROM role_permissions;
DELETE FROM user_roles;
DELETE FROM permissions;
DELETE FROM roles;
