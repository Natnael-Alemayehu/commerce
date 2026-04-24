package rbac

import (
	"context"
	"fmt"

	"starterkit/internal/store"

	"github.com/google/uuid"
)

// Enforcer checks if users have roles or permissions.
type Enforcer interface {
	HasPermission(ctx context.Context, userID string, resource, action string) (bool, error)
	HasRole(ctx context.Context, userID string, roleName string) (bool, error)
	GetUserRoles(ctx context.Context, userID string) ([]store.Role, error)
}

// DBEnforcer implements Enforcer using the database.
type DBEnforcer struct {
	store *store.Store
}

// NewDBEnforcer creates a new DBEnforcer.
func NewDBEnforcer(store *store.Store) *DBEnforcer {
	return &DBEnforcer{store: store}
}

// HasPermission checks if a user has a specific permission.
func (e *DBEnforcer) HasPermission(ctx context.Context, userID string, resource, action string) (bool, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user id: %w", err)
	}

	has, err := e.store.HasPermission(ctx, store.HasPermissionParams{
		UserID:   id,
		Resource: resource,
		Action:   action,
	})
	if err != nil {
		return false, fmt.Errorf("check permission: %w", err)
	}

	return has, nil
}

// HasRole checks if a user has a specific role.
func (e *DBEnforcer) HasRole(ctx context.Context, userID string, roleName string) (bool, error) {
	roles, err := e.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		if role.Name == roleName {
			return true, nil
		}
	}
	return false, nil
}

// GetUserRoles returns all roles for a user.
func (e *DBEnforcer) GetUserRoles(ctx context.Context, userID string) ([]store.Role, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	roles, err := e.store.GetUserRoles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	return roles, nil
}
