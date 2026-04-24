# Adidas-Inspired E-Commerce Platform — Master Build Plan

**Repository:** `https://github.com/Natnael-Alemayehu/commerce`  
**Currency:** ETB (Ethiopian Birr) only  
**Image Storage:** MinIO (S3-compatible, self-hosted)  
**Search:** PostgreSQL Full-Text Search (MVP) → Meilisearch (future upgrade)  
**Payment:** Deferred — orders created as `pending`, payment integration in future round  
**Guest Checkout:** Registration required for MVP. Guest checkout planned for final round.  
**Discounts/Coupons:** Deferred to future round.

---

## 1. Vision

Transform the existing Go starter kit into a production-grade e-commerce API modeled after the Adidas online store experience. The platform supports:

- **Product Catalog:** Hierarchical categories, products with color/size variants, multiple images per variant
- **Persistent Cart:** Cart survives login sessions, validated against live inventory
- **Checkout Flow:** Registration-required checkout, ETB pricing, address selection, order creation
- **Order Management:** Full lifecycle tracking (`pending → confirmed → processing → shipped → delivered → cancelled`)
- **Reviews:** Verified-purchase-only reviews with moderation
- **Wishlist:** Save favorite variants for later
- **Admin Suite:** Product/variant/inventory management, order fulfillment, customer management, review moderation
- **Search:** Full-text product search with faceted filtering

---

## 2. Foundation (What We Keep)

| Component | Status |
|-----------|--------|
| JWT Auth + Argon2id password hashing | Keep as-is |
| RBAC (roles, permissions, middleware) | Keep, extend permissions for commerce |
| Chi router + middleware stack | Keep |
| Structured slog logging + request IDs | Keep |
| Prometheus metrics | Keep |
| SQLC + Goose database layer | Keep |
| Integration test infrastructure | Keep |
| Docker + Docker Compose | Keep, add MinIO service |
| Graceful shutdown | Keep |

---

## 3. Cleanup (What We Remove)

| Component | Action |
|-----------|--------|
| `notes` table | Drop via migration |
| Note-related handlers (`internal/handler/note.go`) | Remove |
| Note-related service (`internal/service/note.go`) | Remove |
| Note-related admin endpoints | Remove from handler |
| Note-related permissions | Remove from RBAC seed data |
| Note-related SQLC queries (`sqlc/notes.sql`) | Remove |
| Note integration tests | Remove |

---

## 4. New Domain Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     E-COMMERCE PLATFORM                      │
├─────────────────────────────────────────────────────────────┤
│  Catalog Domain        │  Cart Domain      │  Order Domain  │
│  ├── categories        │  ├── cart_items   │  ├── orders    │
│  ├── products          │  └── cart totals  │  ├── order_items│
│  ├── product_variants  │                   │  └── shipping  │
│  ├── product_images    │                   │                │
│  └── product_reviews   │                   │                │
├─────────────────────────────────────────────────────────────┤
│  User Domain           │  Inventory Domain │  Admin Domain  │
│  ├── users (extended)  │  └── stock_levels │  ├── products  │
│  ├── addresses         │                   │  ├── orders    │
│  └── wishlist          │                   │  └── customers │
├─────────────────────────────────────────────────────────────┤
│  Infrastructure Domain                                     │
│  ├── MinIO (image storage)                                 │
│  ├── PostgreSQL FTS (search)                               │
│  └── Stripe (payment — future)                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Database Schema

### 5.1 Users & Profiles (extends existing `users` table)

```sql
-- users (extended from existing)
ALTER TABLE users ADD COLUMN avatar_url TEXT;
ALTER TABLE users ADD COLUMN bio TEXT;

-- addresses
CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label VARCHAR(50),                    -- "Home", "Work"
    recipient_name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    street_address TEXT NOT NULL,         -- "123 Main St, Apt 4B"
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'Ethiopia',
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 5.2 Product Catalog

```sql
-- categories (hierarchical: Men > Shoes > Running)
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES categories(id),
    sort_order INT DEFAULT 0,
    image_url TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- products
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id UUID REFERENCES categories(id),
    name VARCHAR(255) NOT NULL,           -- "Ultraboost Light"
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    short_description TEXT,               -- "Energy return running shoes"
    base_price DECIMAL(12,2) NOT NULL,    -- 8500.00 ETB
    compare_at_price DECIMAL(12,2),       -- 12000.00 (for sales display)
    status VARCHAR(20) DEFAULT 'active',  -- active, draft, discontinued
    gender VARCHAR(20),                   -- men, women, unisex, kids
    sport VARCHAR(50),                    -- running, training, soccer, etc.
    brand VARCHAR(50) DEFAULT 'adidas',
    tags TEXT[],                          -- ['new', 'best-seller', 'sustainable']
    weight_g INT,
    material_info TEXT,
    care_instructions TEXT,
    avg_rating DECIMAL(2,1) DEFAULT 0,
    review_count INT DEFAULT 0,
    seo_title VARCHAR(255),
    seo_description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- product_variants (size + color combinations)
CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(100) UNIQUE NOT NULL,     -- "UB-LIGHT-BLK-42"
    variant_name VARCHAR(100),            -- "Core Black / White"
    color_name VARCHAR(50),               -- "Core Black"
    color_hex VARCHAR(7),                 -- "#000000"
    size_label VARCHAR(20),               -- "42", "M", "10.5"
    size_system VARCHAR(10),              -- "EU", "US", "UK"
    price_override DECIMAL(12,2),         -- NULL = use product.base_price
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- product_images
CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id),
    image_url TEXT NOT NULL,              -- MinIO/S3 URL
    alt_text VARCHAR(255),
    sort_order INT DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 5.3 Inventory

```sql
-- inventory (stock per variant)
CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    variant_id UUID NOT NULL UNIQUE REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 0,
    reserved_quantity INT NOT NULL DEFAULT 0,
    low_stock_threshold INT DEFAULT 5,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- stock_movements (audit log)
CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    movement_type VARCHAR(20) NOT NULL,   -- 'in', 'out', 'adjustment', 'return', 'reservation', 'release'
    quantity INT NOT NULL,
    reason TEXT,
    reference_id UUID,                    -- order_id if reservation/release
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 5.4 Cart

```sql
-- cart_items (persistent cart per user)
CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    added_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, variant_id)
);
```

### 5.5 Orders

```sql
-- orders
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    order_number VARCHAR(20) UNIQUE NOT NULL,  -- "ORD-2026-000001"
    status VARCHAR(20) DEFAULT 'pending',      -- pending, confirmed, processing, shipped, delivered, cancelled, refunded
    payment_status VARCHAR(20) DEFAULT 'pending', -- pending, paid, failed, refunded
    shipping_status VARCHAR(20) DEFAULT 'pending',

    -- Financials (all in ETB)
    subtotal DECIMAL(12,2) NOT NULL,
    shipping_cost DECIMAL(12,2) DEFAULT 0,
    tax_amount DECIMAL(12,2) DEFAULT 0,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    total_amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'ETB',

    -- Shipping Address (snapshot at order time)
    shipping_address JSONB NOT NULL,

    -- Payment (Stripe deferred)
    payment_provider VARCHAR(50),         -- 'stripe', 'telebirr', etc.
    payment_reference VARCHAR(255),       -- payment intent ID, transaction ref

    -- Metadata
    notes TEXT,
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- order_items (snapshot of what was purchased)
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES product_variants(id),
    product_name VARCHAR(255) NOT NULL,        -- snapshot
    variant_name VARCHAR(100) NOT NULL,        -- snapshot
    sku VARCHAR(100) NOT NULL,                 -- snapshot
    image_url TEXT,                            -- snapshot
    quantity INT NOT NULL,
    unit_price DECIMAL(12,2) NOT NULL,         -- price at time of purchase
    total_price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 5.6 Reviews & Wishlist

```sql
-- reviews
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id),       -- verify purchase
    rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title VARCHAR(255),
    comment TEXT,
    is_verified_purchase BOOLEAN DEFAULT FALSE,
    status VARCHAR(20) DEFAULT 'pending',      -- pending, approved, rejected
    helpful_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(product_id, user_id)
);

-- wishlist_items
CREATE TABLE wishlist_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    variant_id UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, variant_id)
);
```

---

## 6. API Endpoints

### 6.1 Public Product Catalog

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/v1/products` | No | List products with filters (category, gender, sport, color, size, price_min, price_max, search, sort, page) |
| GET | `/api/v1/products/:slug` | No | Get single product with variants & images |
| GET | `/api/v1/categories` | No | List categories (tree or flat) |
| GET | `/api/v1/categories/:slug` | No | Get category details |
| GET | `/api/v1/categories/:slug/products` | No | Products in category |
| GET | `/api/v1/products/:slug/reviews` | No | List approved reviews |
| GET | `/api/v1/products/featured` | No | Featured/best-seller products |
| GET | `/api/v1/products/related/:slug` | No | Related products in same category |

### 6.2 Cart

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/cart/items` | Bearer | Add item to cart (validates stock) |
| GET | `/api/v1/cart` | Bearer | Get cart with line items, totals, stock status |
| PUT | `/api/v1/cart/items/:id` | Bearer | Update quantity (validates stock) |
| DELETE | `/api/v1/cart/items/:id` | Bearer | Remove item |
| DELETE | `/api/v1/cart` | Bearer | Clear entire cart |

### 6.3 Wishlist

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/wishlist` | Bearer | Add variant to wishlist |
| GET | `/api/v1/wishlist` | Bearer | Get wishlist with product details |
| DELETE | `/api/v1/wishlist/:variant_id` | Bearer | Remove from wishlist |

### 6.4 Checkout & Orders

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/checkout` | Bearer | Create order from cart + selected address |
| GET | `/api/v1/orders` | Bearer | My order history |
| GET | `/api/v1/orders/:id` | Bearer | Order details with items |
| POST | `/api/v1/orders/:id/cancel` | Bearer | Cancel order (only if `pending` or `confirmed`) |
| POST | `/api/v1/orders/:id/reviews` | Bearer | Submit review for purchased product |

### 6.5 User Account

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/v1/me` | Bearer | Get profile |
| PUT | `/api/v1/me` | Bearer | Update profile (name, phone, avatar, bio) |
| GET | `/api/v1/me/addresses` | Bearer | List addresses |
| POST | `/api/v1/me/addresses` | Bearer | Add address |
| PUT | `/api/v1/me/addresses/:id` | Bearer | Update address |
| DELETE | `/api/v1/me/addresses/:id` | Bearer | Delete address |
| PUT | `/api/v1/me/addresses/:id/default` | Bearer | Set default address |

### 6.6 Image Upload (MinIO)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/admin/upload` | Bearer + admin | Get presigned upload URL for image |

### 6.7 Admin

| Method | Endpoint | Auth + Perm | Description |
|--------|----------|-------------|-------------|
| POST | `/api/v1/admin/products` | Bearer + `products:create` | Create product |
| PUT | `/api/v1/admin/products/:id` | Bearer + `products:update` | Update product |
| DELETE | `/api/v1/admin/products/:id` | Bearer + `products:delete` | Soft-delete product |
| POST | `/api/v1/admin/products/:id/variants` | Bearer + `products:update` | Add variant |
| PUT | `/api/v1/admin/variants/:id` | Bearer + `products:update` | Update variant |
| PUT | `/api/v1/admin/inventory/:variant_id` | Bearer + `inventory:update` | Update stock quantity |
| GET | `/api/v1/admin/orders` | Bearer + `orders:list` | List all orders |
| GET | `/api/v1/admin/orders/:id` | Bearer + `orders:read` | Order details |
| PUT | `/api/v1/admin/orders/:id/status` | Bearer + `orders:update` | Update order status |
| GET | `/api/v1/admin/customers` | Bearer + `users:list` | List customers |
| GET | `/api/v1/admin/reviews` | Bearer + `reviews:moderate` | List pending reviews |
| PUT | `/api/v1/admin/reviews/:id` | Bearer + `reviews:moderate` | Approve/reject review |

---

## 7. RBAC Permission Design

| Resource | Actions | Granted To |
|----------|---------|------------|
| `products` | `create`, `update`, `delete` | admin |
| `inventory` | `update`, `read` | admin |
| `orders` | `list`, `read`, `update` | admin |
| `reviews` | `moderate` | admin |
| `upload` | `create` | admin |
| `cart` | `create`, `read`, `update`, `delete` | authenticated user (own) |
| `wishlist` | `create`, `read`, `delete` | authenticated user (own) |
| `orders` (own) | `create`, `read`, `cancel` | authenticated user (own) |
| `reviews` (own) | `create` | authenticated user (own, verified purchase) |
| `addresses` | `create`, `read`, `update`, `delete` | authenticated user (own) |

---

## 8. Search Strategy

### MVP: PostgreSQL Full-Text Search

**Why:** Already running PostgreSQL. Zero new services. Supports:
- `tsvector`/`tsquery` for full-text search
- `websearch_to_tsquery` for Google-like syntax
- Ranking with `ts_rank_cd`
- GIN indexes for fast search

**Implementation:**
- Add `search_vector tsvector` column to `products`
- Populate from `name`, `description`, `tags`
- Create GIN index
- API query: `?q=running shoes` → `websearch_to_tsquery('english', 'running & shoes')`
- Support filtering by category, gender, sport, color, size, price range alongside search

### Future: Meilisearch Upgrade

**Why upgrade:** When faceted search + typo tolerance + instant search-as-you-type become critical (Adidas-level UX).

**Migration path:**
1. Deploy Meilisearch container
2. Build indexer service that syncs product changes to Meilisearch
3. Swap `ProductService.Search()` implementation from SQL to Meilisearch client
4. Keep PostgreSQL as source of truth, Meilisearch as read-optimized search index

---

## 9. Image Storage: MinIO → S3 Migration Path

### Architecture

```
Client (Browser)
    │
    │ GET presigned URL
    ▼
API Server ──► MinIO (self-hosted S3-compatible)
    │               └── bucket: product-images
    │
    └── Store image URL in PostgreSQL (product_images table)
```

### MinIO Setup

- Docker Compose adds MinIO service
- Bucket: `product-images`
- Presigned PUT URLs for upload (time-limited, direct-to-MinIO)
- Public read URLs or presigned GET for private access

### S3 Migration (Future)

**Effort:** Change 3 environment variables:

| Current (MinIO) | Future (AWS S3) |
|-----------------|-----------------|
| `MINIO_ENDPOINT` | `S3_ENDPOINT` |
| `MINIO_ACCESS_KEY` | `AWS_ACCESS_KEY_ID` |
| `MINIO_SECRET_KEY` | `AWS_SECRET_ACCESS_KEY` |

**Code impact:** Zero. The `minio-go` SDK speaks S3 natively. Only the endpoint and credentials change.

---

## 10. Implementation Chunks

### Chunk 1: Foundation Reset
- **Migration:** Drop `notes` table
- **Migration:** Extend `users` with `avatar_url`, `bio`
- **Migration:** Create `addresses` table
- **Update:** RBAC seed data — replace note permissions with commerce permissions
- **Update:** `users.sql` queries for new fields
- **Update:** Auth service — profile update support
- **Update:** Auth handlers — new user endpoints (profile CRUD, addresses CRUD)
- **Update:** Remove note handler, service, and related code
- **Update:** Integration test cleanup (remove note tests)

### Chunk 2: Product Catalog + MinIO
- **Migrations:** `categories`, `products`, `product_variants`, `product_images`
- **Seed data:** Sample categories (Men, Women, Kids, Originals, Running, Training, Soccer, Basketball, Lifestyle)
- **Docker Compose:** Add MinIO service
- **Code:** MinIO client setup (`internal/storage/minio.go`)
- **Code:** Presigned URL generation for image uploads
- **SQLC queries:** Full CRUD for catalog tables
- **Service layer:** Product service with filtering, search, slug generation
- **Handlers:** Public product endpoints (list, get, search, filters)
- **Admin handlers:** Product CRUD, variant management, image upload
- **Tests:** Integration tests for product catalog

### Chunk 3: Inventory System
- **Migrations:** `inventory`, `stock_movements`
- **SQLC queries:** Stock read/update, movement logging
- **Service layer:** Inventory service
  - Stock validation (cart add, checkout)
  - Stock reservation (on order creation)
  - Stock release (on cancellation)
  - Movement audit logging
- **Admin handlers:** Stock management endpoints
- **Integration:** Stock validation in cart service

### Chunk 4: Cart
- **Migration:** `cart_items`
- **SQLC queries:** Cart CRUD
- **Service layer:** Cart service
  - Add item (validate stock, max quantity per item)
  - Update quantity
  - Remove item
  - Clear cart
  - Calculate totals
- **Handlers:** Cart endpoints
- **Integration tests:** Cart flow

### Chunk 5: Checkout & Orders
- **Migrations:** `orders`, `order_items`
- **SQLC queries:** Order CRUD, order item CRUD
- **Service layer:** Order service
  - Checkout: validate cart → validate stock → create order → reserve stock → clear cart
  - Order status transitions (with validation)
  - Order cancellation (release stock)
  - Order history
- **Handlers:** Checkout, order history, order details, cancel
- **Admin handlers:** Order list, order status update
- **Integration tests:** Full checkout flow

### Chunk 6: Reviews & Wishlist
- **Migrations:** `reviews`, `wishlist_items`
- **SQLC queries**
- **Service layer:**
  - Review service (verified purchase check via order_items)
  - Wishlist service
- **Handlers:** Submit review, list reviews, wishlist CRUD
- **Admin handlers:** Review moderation
- **Integration tests:** Review after purchase, wishlist flow

### Chunk 7: Search & Filtering
- **Migration:** Add `search_vector` to `products`, GIN index
- **Trigger:** Auto-update `search_vector` on product insert/update
- **SQLC queries:** Full-text search with filtering
- **Service layer:** Search service
- **Handlers:** Search endpoint with query params
- **Integration tests:** Search accuracy

### Chunk 8: Admin Polish & End-to-End Tests
- Complete admin endpoint coverage
- Seed script: Sample products with variants and images
- Integration tests:
  - Full purchase flow: browse → search → cart → checkout → order
  - Stock reservation and release on cancel
  - Review submission after delivery
  - Admin product/order management
  - RBAC: customer vs admin access control

---

## 11. Key Design Decisions

1. **Soft deletes for products** — Admin can restore discontinued products. Variants can be deactivated without deleting.
2. **Price snapshot on orders** — `order_items` stores `unit_price` at purchase time. Future product price changes don't affect historical orders.
3. **Stock reservation pattern** — Stock is reserved when order is created, released if order is cancelled. Prevents overselling.
4. **Variant-centric inventory** — Every size/color combo has its own SKU and stock level. This is how Adidas works.
5. **Address snapshot in orders** — `shipping_address` is JSONB, capturing full address at time of order. User can update their address book without affecting past orders.
6. **Slug-based product URLs** — Human-readable URLs like `/api/v1/products/ultraboost-light`.
7. **Verified purchase reviews** — Only users with a `delivered` order containing the product can leave verified reviews.
8. **ETB currency hardcoded** — No multi-currency complexity in MVP. All prices stored and displayed in Ethiopian Birr.
9. **MinIO for images** — Self-hosted S3-compatible storage. Zero-cost for development. Easy migration to AWS S3, Cloudflare R2, or DigitalOcean Spaces later.
10. **Payment deferred** — Orders created with `payment_status = 'pending'`. Payment integration (Stripe, Telebirr, Chapa) will be a dedicated future chunk.

---

## 12. Deferred to Future Rounds

| Feature | Round |
|---------|-------|
| Payment integration (Stripe, Telebirr, Chapa) | Round 2 |
| Guest checkout | Final round |
| Discounts / coupon codes | Round 2 |
| Multi-currency support | Future |
| Meilisearch upgrade | Future (when search UX demands it) |
| Email notifications (order confirmation, shipping) | Round 2 |
| Order tracking / logistics integration | Round 2 |
| Product recommendations / ML | Future |
| Real-time inventory (WebSocket) | Future |
| Mobile app API optimization | Future |

---

## 13. Docker Compose Services (MVP)

```yaml
services:
  db:
    image: postgres:16-alpine
    # ... existing config

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: commerce
      MINIO_ROOT_PASSWORD: commerce123
    volumes:
      - minio_data:/data

  api:
    build: .
    environment:
      DATABASE_URL: postgres://...
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: commerce
      MINIO_SECRET_KEY: commerce123
      MINIO_BUCKET: product-images
      MINIO_USE_SSL: "false"
      # ... other env vars
```

---

## 14. Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENV` | No | `development` | Environment mode |
| `PORT` | No | `8080` | HTTP server port |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `JWT_SECRET` | Yes | — | JWT signing secret |
| `JWT_ACCESS_TTL` | No | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | No | `168h` | Refresh token lifetime |
| `MINIO_ENDPOINT` | Yes | — | MinIO host:port |
| `MINIO_ACCESS_KEY` | Yes | — | MinIO access key |
| `MINIO_SECRET_KEY` | Yes | — | MinIO secret key |
| `MINIO_BUCKET` | No | `product-images` | MinIO bucket name |
| `MINIO_USE_SSL` | No | `false` | Use HTTPS for MinIO |
| `ARGON2_*` | No | sensible defaults | Argon2id parameters |

---

*Plan finalized. Ready for execution upon confirmation.*
