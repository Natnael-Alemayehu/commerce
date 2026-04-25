package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"starterkit/internal/config"
	"starterkit/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestDB holds test database resources.
type TestDB struct {
	Pool  *pgxpool.Pool
	Store *store.Store
}

// NewTestDB creates a new test database connection.
// Skips the test if TEST_DATABASE_URL is not set and cannot connect to default.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		// Try default test database
		databaseURL = "postgres://starterkit:starterkit@localhost:5432/starterkit_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Cannot ping test database: %v", err)
	}

	// Run migrations
	if err := runMigrations(databaseURL); err != nil {
		pool.Close()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	db := store.NewStore(pool)

	return &TestDB{
		Pool:  pool,
		Store: db,
	}
}

// Cleanup truncates all tables for a clean state.
func (td *TestDB) Cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	_, err := td.Pool.Exec(ctx, `
		TRUNCATE TABLE product_images, product_variants, products, categories, addresses, refresh_tokens, user_roles, role_permissions, permissions, roles, users RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("Failed to cleanup test database: %v", err)
	}

	// Re-seed RBAC data after cleanup
	_, err = td.Pool.Exec(ctx, `
		INSERT INTO roles (name, description) VALUES
			('admin', 'System administrator with full access'),
			('user', 'Standard user with limited access');

		INSERT INTO permissions (name, resource, action) VALUES
			('users:list', 'users', 'list'),
			('users:read', 'users', 'read'),
			('users:update', 'users', 'update'),
			('users:delete', 'users', 'delete'),
			('products:create', 'products', 'create'),
			('products:update', 'products', 'update'),
			('products:delete', 'products', 'delete'),
			('inventory:update', 'inventory', 'update'),
			('inventory:read', 'inventory', 'read'),
			('orders:list', 'orders', 'list'),
			('orders:read', 'orders', 'read'),
			('orders:update', 'orders', 'update'),
			('reviews:moderate', 'reviews', 'moderate'),
			('upload:create', 'upload', 'create');

		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.name = 'admin';

		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
		WHERE r.name = 'user' AND p.name IN (
			'users:read', 'users:update', 'users:delete'
		);

		-- Re-seed categories
		INSERT INTO categories (name, slug, description, sort_order) VALUES
			('Men', 'men', 'Men collection', 1),
			('Women', 'women', 'Women collection', 2),
			('Kids', 'kids', 'Kids collection', 3),
			('Originals', 'originals', 'adidas Originals', 4),
			('Running', 'running', 'Running shoes and apparel', 5),
			('Training', 'training', 'Training and gym', 6),
			('Soccer', 'soccer', 'Soccer and football', 7),
			('Basketball', 'basketball', 'Basketball gear', 8),
			('Lifestyle', 'lifestyle', 'Lifestyle and casual', 9);
	`)
	if err != nil {
		t.Fatalf("Failed to re-seed RBAC data: %v", err)
	}
}

// Close closes the database pool.
func (td *TestDB) Close() {
	td.Pool.Close()
}

func runMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(db, "../../migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// TestConfig returns a test configuration.
func TestConfig() *config.Config {
	return &config.Config{
		Environment:    "test",
		Port:           "8080",
		DatabaseURL:    os.Getenv("TEST_DATABASE_URL"),
		JWTSecret:      "test-secret-key-that-is-32-bytes!",
		JWTAccessTTL:   15 * time.Minute,
		JWTRefreshTTL:  168 * time.Hour,
		Argon2Time:     1,
		Argon2Memory:   65536,
		Argon2Threads:  4,
		Argon2KeyLen:   32,
		Argon2SaltLen:  16,
		MinioEndpoint:  "localhost:9000",
		MinioAccessKey: "test",
		MinioSecretKey: "test",
		MinioBucket:    "product-images",
		MinioUseSSL:    false,
	}
}
