package server

import (
	"log/slog"
	"net/http"
	"time"

	"starterkit/internal/auth"
	"starterkit/internal/config"
	"starterkit/internal/handler"
	"starterkit/internal/middleware"
	"starterkit/internal/rbac"
	"starterkit/internal/service"
	"starterkit/internal/storage"
	"starterkit/internal/store"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "starterkit/docs"
)

// New creates and configures a new HTTP server router.
func New(db *store.Store, passwordHasher *auth.PasswordHasher, jwtManager *auth.JWTManager, cfg *config.Config) http.Handler {
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
	productService := service.NewProductService(db)
	enforcer := rbac.NewDBEnforcer(db)

	// Storage
	minioClient, err := storage.NewMinIOClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		// Log error but don't fail startup - image uploads will fail gracefully
		slog.Default().Error("failed to initialize minio client", "error", err)
		minioClient = nil
	}

	// Auth Routes (no auth middleware needed for some)
	authHandler := handler.NewAuthHandler(authService)
	authHandler.RegisterRoutes(r)

	// User Routes (profile, addresses)
	userHandler := handler.NewUserHandler(authService, db)
	userHandler.RegisterRoutes(r)

	// Public Product Catalog Routes
	productHandler := handler.NewProductHandler(productService)
	productHandler.RegisterRoutes(r)

	// Admin Routes (require auth + admin permission)
	authMiddleware := middleware.Auth(jwtManager)
	adminHandler := handler.NewAdminHandler(db)
	adminProductHandler := handler.NewAdminProductHandler(productService, minioClient)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "users", "list"))
		r.Get("/api/v1/admin/users", adminHandler.ListUsers)
	})

	// Admin Product Routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "products", "create"))
		r.Post("/api/v1/admin/products", adminProductHandler.CreateProduct)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "products", "update"))
		r.Put("/api/v1/admin/products/{id}", adminProductHandler.UpdateProduct)
		r.Post("/api/v1/admin/products/{id}/variants", adminProductHandler.CreateVariant)
		r.Put("/api/v1/admin/variants/{id}", adminProductHandler.UpdateVariant)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "products", "delete"))
		r.Delete("/api/v1/admin/products/{id}", adminProductHandler.DeleteProduct)
		r.Delete("/api/v1/admin/variants/{id}", adminProductHandler.DeleteVariant)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(middleware.RequirePermission(enforcer, "upload", "create"))
		r.Post("/api/v1/admin/upload", adminProductHandler.PresignedUploadURL)
	})

	return r
}
