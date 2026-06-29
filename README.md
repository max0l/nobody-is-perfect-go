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

## Running The Server

Run with defaults:

```sh
go run .
```

The server listens on `0.0.0.0:8080` by default. OpenAPI docs are available at:

```text
http://localhost:8080/swagger/index.html
```

Build and run the binary:

```sh
go build -o nobody-is-perfect-go .
./nobody-is-perfect-go
```

Configure the server with environment variables:

```sh
NIP_HOST=127.0.0.1 NIP_PORT=3000 go run .
```

Or copy `.env.example` and load it with your shell or process manager:

```sh
set -a
. ./.env
set +a
go run .
```

## Configuration

| Environment variable | Default | Description |
| --- | --- | --- |
| `NIP_HOST` | `0.0.0.0` | Host/interface the HTTP server binds to. |
| `NIP_PORT` | `8080` | Port the HTTP server listens on. |
| `NIP_API_BASE_URL` | Derived from host/port, normally `http://localhost:8080` | Public API base URL written into the OpenAPI server config for request validation. Set this when Docker/proxy host or port differs from the internal bind address. |
| `NIP_MAX_CONCURRENT_GAMES` | `100` | Maximum number of active games. Creating another game returns `403 Forbidden`. |
| `NIP_WORDLIST_PATH` | `words.txt` | Path to the word list used for generated game IDs. |

## Docker

Build the image locally:

```sh
docker build -t nobody-is-perfect-go .
```

Run it with defaults:

```sh
docker run --rm -p 8080:8080 nobody-is-perfect-go
```

Override configuration with environment variables:

```sh
docker run --rm -p 3000:3000 \
  -e NIP_PORT=3000 \
  -e NIP_API_BASE_URL=http://localhost:3000 \
  -e NIP_MAX_CONCURRENT_GAMES=100 \
  nobody-is-perfect-go
```

The published image is available from GitHub Container Registry:

```sh
docker pull ghcr.io/max0l/nobody-is-perfect-go:latest
docker run --rm -p 8080:8080 ghcr.io/max0l/nobody-is-perfect-go:latest
```

Images are published only for pushes to `main` and Git tags matching `v*`.

Run with Docker Compose:

```sh
docker compose up --build
```

Compose uses the same environment variables and defaults as the server. Override them inline or through a local `.env` file:

```sh
NIP_HOST=127.0.0.1 NIP_PORT=3000 NIP_API_BASE_URL=http://127.0.0.1:3000 docker compose up --build
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
