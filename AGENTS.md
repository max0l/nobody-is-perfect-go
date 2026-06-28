# Repository Guidelines

## Code Quality
- Keep changes small and focused. Avoid unrelated cleanup in feature commits.
- Put domain/game rules in `game`; keep HTTP request/response mapping in `api`.
- Preserve secure session-cookie auth on protected endpoints. Protected endpoints must require the `session` cookie.
- Add or update tests for behavior changes, especially game-state transitions and authorization rules.
- Prefer explicit state checks over implicit assumptions. Mutating endpoints should verify both phase and actor.
- Use `zerolog` for application logging; do not add standard-library `log` call sites.

## OpenAPI Workflow
- `api.yaml` is the source of truth for HTTP contracts.
- Do not edit `api/api-server.gen.go` or `api/api-types.gen.go` manually.
- After changing `api.yaml`, run `task generate` or `go generate ./api`.

## Required Checks
- Format Go code with `gofmt`.
- Run `go test ./...` before committing.
- Run `task verify` before committing when `go-task` is installed.

## Git Hooks
- Install hooks with `task install-hooks`.
- The pre-commit hook delegates to `task verify`.
- If `task` is not installed, install it with `go install github.com/go-task/task/v3/cmd/task@latest`.
