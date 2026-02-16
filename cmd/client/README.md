# CLI client (`cmd/client`)

Command-line client for `go-data-keeper`. The client:

- authenticates (register/login) and persists session state locally;
- encrypts user data locally before sending it to the server;
- keeps a local vault of encrypted items and supports syncing changes.

## Build and run

From the repository root:

```bash
go run ./cmd/client --help
```

Build a binary:

```bash
go build -o client ./cmd/client
./client --help
```

## Global flags

The CLI defines these persistent flags:

- `--server` — server URL (default: `http://127.0.0.1:8080`)
- `--state` — path to the client state file (`state.json`)
- `--vault` — path to the local vault file (`vault.json`)

## Local state and paths (cross-platform)

Default paths are computed using `os.UserConfigDir()` with fallback to `os.UserHomeDir()`, and stored under a `gophkeeper/` subdirectory:

- `.../gophkeeper/state.json`
- `.../gophkeeper/vault.json`

You can override the locations via `--state` / `--vault` (useful for testing and isolating environments).

## Commands (high level)

Use `--help` on each command for exact flags/arguments. Below is a high-level overview (without duplicating API docs).

- `register` — register a new user (on success, triggers `sync`).
- `login` — log in an existing user (on success, triggers `sync`).
- `set` — create a new item or update an existing one via `--id` (payload is encrypted on the client).
  - Supported types: `login/pass`, `text`, `binary`, `card`.
  - For `binary`, use `--file`; for other types, use `--data`.
  - Optional metadata: `--meta key=value` (repeatable) and `--meta-text` (free-form text).
- `get` — fetch an item by `--id` and decrypt it locally (requires `--password`).
  - For `binary`, `--out` is required (output file path).
- `list` — print items from the local vault.
  - Flags: `--type`, `--deleted`, `--all`.
  - Note: **`--all` takes precedence over `--deleted`** (if both are set, the result is “all items including deleted”; the `--deleted` filter is effectively ignored).
- `delete` — delete an item by `--id` (soft delete / tombstone; encrypted payload is cleared in the local vault).
- `sync` — manually sync changes from the server starting at `LastSyncedRev` stored in `state.json`.
- `version` — print build version/date/commit (from `internal/buildinfo`).

## Manual testing (scripts)

The `test_scripts/` folder contains materials for manual CLI testing:

- `test_scripts/cli_mock_test.ps1` — prepares a test run and builds a reusable CLI binary into `tmp/`
- `test_scripts/hand_test.txt` — step-by-step commands (happy-path + negative checks)

A practical pattern is to use separate `--state`/`--vault` paths for tests, so you don’t mix test data with real local state.
