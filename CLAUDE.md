# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Monorepo Layout

```
services/
  api/       → Go REST API (primary service)
  ai/        → Python AI/RAG service
```

Run commands from the service directory (e.g. `cd services/api`), or use the root `Makefile`.

## Commands

```bash
# From repo root
make run-api
make run-ai
make test-api
docker-compose up -d

# From services/api/
go run cmd/app/main.go
go test ./...
go test ./internal/usecase/shortcode -run TestGenerateShortUrl
go mod download
```

## Architecture (services/api)

Clean Architecture with 4 layers:

```
services/api/
  cmd/app/main.go          → DI container, server setup, graceful shutdown
  internal/
    entity/                → Domain models (Link, File, User)
    usecase/               → Business logic, interface definitions
    usecase/shortcode/     → Short code generation (base58 + sha256)
    controller/http/       → Gin handlers, DTOs
    repository/
      mongodb/             → Link persistence
      redis/               → Link caching
      s3/                  → File storage
    service/
      auth/cognito/        → AWS Cognito auth adapter
      ai/                  → AI service HTTP client
  pkg/
    config/                → Env-based configuration
    errors/                → Domain error types
```

**Dependency flow**: `controller → usecase → repository/service` (via interfaces in `usecase/interfaces.go`)

## Key Patterns

- Interfaces defined in `internal/usecase/interfaces.go` and `internal/usecase/storage.go`
- Error types in `pkg/errors/errors.go` with `Is*` helper functions
- Cache-aside pattern: check Redis first, fallback to MongoDB, then populate cache
- Retry logic with configurable attempts for MongoDB/Redis connections
- External service adapters (Cognito, AI) live in `internal/service/`, not `internal/repository/`

## Configuration

Environment variables loaded via `services/api/pkg/config/config.go`. See `.env.sample` for all options.
