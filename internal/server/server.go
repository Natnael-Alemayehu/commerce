package server

import (
	"net/http"
	"time"

	"starterkit/internal/auth"
	"starterkit/internal/handler"
	"starterkit/internal/middleware"
	"starterkit/internal/rbac"
	"starterkit/internal/service"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "starterkit/docs"
)

// New creates and configures a new HTTP server router.
func New(db *store.Store, passwordHasher *auth.PasswordHasher, jwtManager *auth.JWTManager) http.Handler {
	r := chi.NewRouter()

	// Middleware stack (order matters - first executed is last in this list)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Metrics)
	r.Use(middleware.RequestLogger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.RateLimit(100, 1*time.Minute))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","checks":{"database":"down"}}`))
			return
		}

		w.Write([]byte(`{"status":"healthy","checks":{"database":"up"}}`))
	})

	// Metrics
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Services
	authService := service.NewAuthService(db, passwordHasher, jwtManager)
	enforcer := rbac.NewDBEnforcer(db)
	noteService := service.NewNoteService(db, enforcer)

	// Auth Routes (no auth middleware needed for some)
	authHandler := handler.NewAuthHandler(authService)
	authHandler.RegisterRoutes(r)

	// Note Routes (require auth)
	authMiddleware := middleware.Auth(jwtManager)
	noteHandler := handler.NewNoteHandler(noteService)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "create"))
		r.Post("/api/v1/notes", noteHandler.Create)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "list"))
		r.Get("/api/v1/notes", noteHandler.List)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "read"))
		r.Get("/api/v1/notes/{id}", noteHandler.Get)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "update"))
		r.Put("/api/v1/notes/{id}", noteHandler.Update)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "delete"))
		r.Delete("/api/v1/notes/{id}", noteHandler.Delete)
	})

	// Admin Routes (require auth + admin permission)
	adminHandler := handler.NewAdminHandler(noteService)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "users", "list"))
		r.Get("/api/v1/admin/users", adminHandler.ListUsers)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "list:all"))
		r.Get("/api/v1/admin/notes", adminHandler.ListAllNotes)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "list:deleted"))
		r.Get("/api/v1/admin/notes/deleted", adminHandler.ListDeletedNotes)
	})
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "notes", "restore"))
		r.Post("/api/v1/admin/notes/{id}/restore", adminHandler.RestoreNote)
	})

	return r
}
