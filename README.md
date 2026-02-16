# go-data-keeper

`go-data-keeper` is a Go client–server project for securely storing user data. The CLI client encrypts data locally and talks to an HTTP API; the server stores encrypted items and supports syncing changes.

## Architecture (high level)

- **Client / Server split**
  - `cmd/server` — HTTP server.
  - `cmd/client` — CLI client built with `cobra`.
- **Server layers**
  - `internal/handler` — HTTP handlers (routing layer).
  - `internal/service` — business logic.
  - `internal/repository` — database access.
- **Shared transport contracts**
  - `pkg/contract` — DTOs shared between client and server.
- **Swagger/OpenAPI artifacts**
  - `docs/` — generated Swagger files.

## Quick start

### Run the server

1. Create `.env` from `.env.EXAMPLE`.
2. Start the server from the repository root:

```bash
go run ./cmd/server
```

### Run the CLI client

Run directly:

```bash
go run ./cmd/client --help
```

Or build a binary:

```bash
go build -o client ./cmd/client
./client --help
```

## Configuration overview

### Server

The server loads `.env` (if present) and reads configuration from environment variables:

- `RUN_ADDRESS` — server listen address (for example `:8080`)
- `DATABASE_URI` — PostgreSQL DSN
- `JWT_SECRET` — JWT signing secret (must be at least 32 bytes)

It also supports flags (see `internal/config`): `-a` (address), `-db` (database uri).

### Client (local state)

The client persists local files:

- **state**: `state.json` (JWT, KDF parameters, `LastSyncedRev`, etc.)
- **vault**: `vault.json` (local store of encrypted items + tombstones for deleted items)

Default locations are computed using `os.UserConfigDir()` with fallback to `os.UserHomeDir()` for cross-platform behavior (Windows/Linux/macOS). You can override paths via `--state` and `--vault`.

## API documentation (Swagger)

- **Swagger UI route**: `/swagger/*`
  - Typically: `http://127.0.0.1:8080/swagger/index.html`
- API details are documented via Swagger UI (this README does not duplicate endpoint docs).

### Regenerate Swagger docs

Swagger docs are generated with `swag`:

```bash
swag init -g cmd/server/main.go
```

## Manual testing

The repository contains scripts/notes for manual testing:

- `test_scripts/cli_mock_test.ps1` — prepares a test run and builds a CLI binary into `tmp/`
- `test_scripts/hand_test.txt` — step-by-step commands (happy-path + negative checks)

## Godoc (package documentation UI)

To browse package docs locally:

```bash
godoc -http=:6060
```

Then open (example): `http://127.0.0.1:6060/pkg/github.com/makimaki04/go-data-keeper.git/`.

## Component docs

- Server: `cmd/server/README.md`
- Client: `cmd/client/README.md`