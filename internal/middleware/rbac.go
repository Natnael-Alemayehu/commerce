package middleware

import (
	"net/http"

	"starterkit/internal/logger"
	"starterkit/internal/rbac"
)

// RequirePermission creates middleware that checks if the user has a specific permission.
func RequirePermission(enforcer rbac.Enforcer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				logger.FromContext(r.Context()).Warn("missing user id for permission check")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"authentication required"}}`))
				return
			}

			hasPerm, err := enforcer.HasPermission(r.Context(), userID, resource, action)
			if err != nil {
				logger.FromContext(r.Context()).Error("permission check failed", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"failed to check permissions"}}`))
				return
			}

			if !hasPerm {
				logger.FromContext(r.Context()).Warn("permission denied",
					"user_id", userID,
					"resource", resource,
					"action", action,
				)
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"you do not have permission to perform this action"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that checks if the user has a specific role.
func RequireRole(enforcer rbac.Enforcer, roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				logger.FromContext(r.Context()).Warn("missing user id for role check")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"authentication required"}}`))
				return
			}

			hasRole, err := enforcer.HasRole(r.Context(), userID, roleName)
			if err != nil {
				logger.FromContext(r.Context()).Error("role check failed", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"failed to check role"}}`))
				return
			}

			if !hasRole {
				logger.FromContext(r.Context()).Warn("role denied",
					"user_id", userID,
					"role", roleName,
				)
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"you do not have the required role"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
