package middleware

import (
	"context"
	"net/http"
	"strings"

	"starterkit/internal/auth"
	"starterkit/internal/logger"
)

type contextKey string

const userIDKey contextKey = "user_id"

// Auth creates a middleware that validates JWT access tokens.
func Auth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.FromContext(r.Context()).Warn("missing authorization header")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"missing authorization header"}}`))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				logger.FromContext(r.Context()).Warn("invalid authorization header format")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"invalid authorization header format"}}`))
				return
			}

			token := parts[1]
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				logger.FromContext(r.Context()).Warn("invalid token", "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"invalid or expired token"}}`))
				return
			}

			ctx := WithUserID(r.Context(), claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext retrieves the user ID from the context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
