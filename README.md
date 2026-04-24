# Go Web Service Starter Kit

A production-ready Go web service starter kit with JWT authentication, Argon2id password hashing, SQLC for type-safe database queries, Goose migrations, and Chi router.

## Features

- **Authentication**: JWT access & refresh tokens with secure Argon2id password hashing
- **Token Management**: Refresh token persistence with revocation (logout/logout-all)
- **RBAC**: Full role-based access control with roles, permissions, and middleware enforcement
- **Note CRUD**: Complete note management with soft deletes and admin oversight
- **Database**: PostgreSQL with SQLC (type-safe queries) and Goose (migrations)
- **Router**: Chi with middleware stack (logging, CORS, rate limiting, recovery, metrics)
- **API Docs**: Swagger UI auto-generated from Go comments
- **Structured Logging**: `log/slog` with request IDs and context propagation
- **Metrics**: Prometheus metrics for request count, duration, and in-flight requests
- **Graceful Shutdown**: SIGINT/SIGTERM handling with connection draining
- **Integration Tests**: Full end-to-end tests with real database

## Project Structure

```
.
├── cmd/api/              # Application entry point
├── internal/
│   ├── auth/             # JWT + Argon2id password hashing
│   ├── config/           # ENV-based configuration
│   ├── handler/          # HTTP handlers (auth, notes, admin)
│   ├── logger/           # slog setup with request IDs
│   ├── middleware/       # Custom Chi middleware (auth, RBAC, metrics)
│   ├── model/            # Request/response DTOs
│   ├── rbac/             # RBAC enforcer interface + implementation
│   ├── server/           # Router setup and middleware stack
│   ├── service/          # Business logic layer
│   ├── store/            # Data access (SQLC-generated + wrappers)
│   └── testutil/         # Test helpers and database setup
├── migrations/           # Goose migration files
├── sqlc/                 # SQLC query definitions
├── docs/                 # Swaggo-generated OpenAPI docs
├── docker-compose.yml    # Postgres + app services
├── Dockerfile            # Multi-stage build
├── Makefile              # Dev commands
└── .env.example          # Environment variables template
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Make

### Local Development

1. **Copy environment variables:**
   ```bash
   cp .env.example .env
   ```

2. **Start PostgreSQL:**
   ```bash
   docker compose up -d db
   ```

3. **Run migrations:**
   ```bash
   make migrate-up
   ```

4. **Start the server:**
   ```bash
   make run
   ```

### Docker Compose (Full Stack)

```bash
docker compose up --build
```

This starts both PostgreSQL and the API server with hot reload.

## API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/health` | Health check (with DB status) | No |
| GET | `/metrics` | Prometheus metrics | No |
| GET | `/swagger/*` | Swagger UI | No |
| **Auth** ||||
| POST | `/api/v1/auth/register` | Register new user | No |
| POST | `/api/v1/auth/login` | Login user | No |
| POST | `/api/v1/auth/refresh` | Refresh access token | No |
| POST | `/api/v1/auth/logout` | Logout (revoke refresh token) | No |
| POST | `/api/v1/auth/logout-all` | Logout from all devices | Bearer |
| GET | `/api/v1/auth/me` | Get current user | Bearer |
| DELETE | `/api/v1/auth/me` | Delete account | Bearer |
| **Notes** ||||
| POST | `/api/v1/notes` | Create note | Bearer |
| GET | `/api/v1/notes` | List my notes | Bearer |
| GET | `/api/v1/notes/:id` | Get single note | Bearer |
| PUT | `/api/v1/notes/:id` | Update note | Bearer |
| DELETE | `/api/v1/notes/:id` | Soft-delete note | Bearer |
| **Admin** ||||
| GET | `/api/v1/admin/users` | List all users | Bearer + admin |
| GET | `/api/v1/admin/notes` | List all notes | Bearer + admin |
| GET | `/api/v1/admin/notes/deleted` | List deleted notes | Bearer + admin |
| POST | `/api/v1/admin/notes/:id/restore` | Restore deleted note | Bearer + admin |

## Configuration

All configuration is loaded from environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ENV` | `development` | Environment (development/production) |
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | *required* | PostgreSQL connection string |
| `JWT_SECRET` | *required* | Secret key for JWT signing |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (7 days) |
| `ARGON2_*` | sensible defaults | Argon2id parameters |

## Development Commands

```bash
make run              # Run the server locally
make build            # Build binary
make test             # Run unit tests
make test-integration # Run integration tests (requires test database)
make deps             # Install dependencies
make migrate-up       # Run migrations
make migrate-down     # Rollback migrations
make sqlc-generate    # Regenerate SQLC code
make swagger          # Regenerate Swagger docs
```

### Integration Tests

Integration tests require a test database. By default they try to connect to:
`postgres://starterkit:starterkit@localhost:5432/starterkit_test?sslmode=disable`

Set `TEST_DATABASE_URL` env var to override:
```bash
export TEST_DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable"
make test-integration
```

## Architecture

### Layered Architecture

1. **Handler** (`internal/handler`): HTTP layer, JSON serialization, input validation
2. **Service** (`internal/service`): Business logic, orchestration
3. **Store** (`internal/store`): Data access, SQLC-generated queries

### Authentication Flow

1. **Register/Login**: Server hashes password with Argon2id, generates JWT pair (access + refresh), stores refresh token hash in DB
2. **Access Protected Routes**: Client sends `Authorization: Bearer <access_token>`, server validates JWT
3. **Refresh**: Client sends refresh token, server validates JWT, checks DB for revocation, issues new pair and revokes old token

### RBAC Flow

1. **Roles & Permissions**: Stored in database tables (`roles`, `permissions`, `role_permissions`)
2. **User Roles**: Users get `user` role on registration; admins can have additional roles via `user_roles`
3. **Permission Check**: Middleware checks `HasPermission(userID, resource, action)` before allowing access
4. **Ownership vs Admin**: Service layer checks note ownership first, then falls back to admin permission (`:all` actions)

## Security

- **Passwords**: Argon2id with configurable time, memory, threads
- **Tokens**: Short-lived access tokens (15 min), longer refresh tokens (7 days)
- **Rate Limiting**: 100 requests/minute per IP
- **CORS**: Configurable via middleware
- **Input Validation**: `go-playground/validator` with struct tags

## License

MIT
