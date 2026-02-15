# Server (`cmd/server`)

HTTP server that provides an API for user registration/login and for working with encrypted items. The server also exposes an endpoint to fetch changes for client synchronization.

## Run locally

From the repository root:

```bash
go run ./cmd/server
```

## Configuration

The server reads configuration from environment variables and (if present) the `.env` file in the repository root.

Environment variables:

- `RUN_ADDRESS` — server listen address (for example `:8080`)
- `DATABASE_URI` — PostgreSQL connection string (DSN)
- `JWT_SECRET` — JWT signing secret (must be at least 32 bytes)

Startup flags:

- `-a` — address
- `-db` — database uri

See `.env.EXAMPLE` for a minimal example.

## Database and migrations

- PostgreSQL is used (configured via `DATABASE_URI`).
- Migrations are **embedded** (via `embed`) and are applied automatically on startup.
  - Migration files: `internal/migrations/migration_files/*.sql`

## Swagger / OpenAPI

- **Swagger UI route**: `/swagger/*`
  - Typically: `http://127.0.0.1:8080/swagger/index.html`
- API details are documented via Swagger UI (this README does not duplicate endpoint docs).

## Logging / observability

Logging is based on `zap` and is configured via:

- `configs/logger.json`