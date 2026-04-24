# ---------- Build stage ----------
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/api ./cmd/api

# ---------- Development stage ----------
FROM golang:1.23-alpine AS dev

WORKDIR /app

RUN apk add --no-cache git make

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/pressly/goose/v3/cmd/goose@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "run", "./cmd/api"]

# ---------- Production stage ----------
FROM gcr.io/distroless/static-debian12:nonroot AS production

WORKDIR /app

COPY --from=builder /app/bin/api /app/api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
