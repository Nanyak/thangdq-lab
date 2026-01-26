# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the application
go run cmd/app/main.go

# Run all tests
go test ./...

# Run a specific test
go test ./internal/usecase/shortcode -run TestGenerateShortUrl

# Download dependencies
go mod download

# Docker (requires MongoDB and Redis)
docker-compose up -d
```

## Architecture

Clean Architecture with 4 layers:

```
cmd/app/main.go          → DI container, server setup, graceful shutdown
internal/
  entity/                → Domain models (Link, File)
  usecase/               → Business logic, interface definitions
  usecase/shortcode/     → Short code generation (base58 + sha256)
  controller/http/       → Gin handlers, DTOs
  repository/
    mongodb/             → Link persistence
    redis/               → Link caching
    s3/                  → File storage
pkg/
  config/                → Env-based configuration
  errors/                → Domain error types
```

**Dependency flow**: `controller → usecase → repository` (via interfaces in `usecase/interfaces.go`)

## Key Patterns

- Interfaces defined in `internal/usecase/interfaces.go` and `internal/usecase/storage.go`
- Error types in `pkg/errors/errors.go` with `Is*` helper functions
- Cache-aside pattern: check Redis first, fallback to MongoDB, then populate cache
- Retry logic with configurable attempts for MongoDB/Redis connections

## Configuration

Environment variables loaded via `pkg/config/config.go`. See `.env.sample` for all options.
