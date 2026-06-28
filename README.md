# nobody-is-perfect-go

Backend for a digital helper around the physical game "Nobody is Perfect".

The service manages users, games, player order, round phases, answers, voting, reveal state, player presence, and secure session-cookie authentication.

## Requirements

- Go `1.26.1`
- Optional but recommended: [`go-task`](https://taskfile.dev/)

Install `go-task`:

```sh
go install github.com/go-task/task/v3/cmd/task@latest
```

## Common Commands

```sh
task generate
task test
task verify
task build
```

Without `task`:

```sh
go generate ./api
go test ./...
go build ./...
```

## Git Hooks

Install the tracked pre-commit hook:

```sh
task install-hooks
```

The hook runs `task verify`, which checks formatting, regenerates OpenAPI code, verifies generated files are current, and runs tests.

## OpenAPI Code Generation

`api.yaml` is the source of truth for HTTP endpoints and schemas.

Generated files must not be edited manually:

- `api/api-server.gen.go`
- `api/api-types.gen.go`

After changing `api.yaml`, run:

```sh
task generate
```

## Authentication

The app uses a secure `session` cookie for protected endpoints.

Session cookies are issued on user creation with:

- `HttpOnly`
- `Secure`
- `SameSite=Strict`
- `Path=/`

Game endpoints that require authentication are protected by the OpenAPI `sessionCookie` security scheme and server-side session middleware.

## Game Flow

- A user creates a game and becomes the game creator.
- The creator can set player order and start the game.
- Each round has a round master based on play order.
- Players submit answers during the answering phase.
- The round master sees scrambled answers with authors; other players see scrambled anonymous answers.
- The creator or round master starts voting manually.
- The round master does not vote.
- Non-master players vote secretly until reveal.
- The creator or round master reveals votes.
- The next round rotates the round master by play order.

## Presence

Players should call the ping endpoint roughly every 5 seconds:

```http
POST /api/game/{gameId}/ping
```

Players are considered offline after missing pings for a short timeout. A game is discarded after all players have been offline for 60 seconds.

The game status and play-order responses include each player's `online` indicator.

## Code Guidelines

See `AGENTS.md` for repository-specific coding and review guidelines.
