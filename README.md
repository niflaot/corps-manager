# discord-bot v1.0.0

[![Version](https://img.shields.io/badge/version-v1.0.0-5865F2.svg)](https://github.com/pixelados-net/discord-bot/releases/tag/v1.0.0)
[![CI](https://github.com/pixelados-net/discord-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/pixelados-net/discord-bot/actions/workflows/ci.yml)
[![Package](https://github.com/pixelados-net/discord-bot/actions/workflows/package.yml/badge.svg)](https://github.com/pixelados-net/discord-bot/actions/workflows/package.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/pixelados-net/discord-bot.svg)](https://pkg.go.dev/github.com/pixelados-net/discord-bot)

`discord-bot` is a production-oriented Go boilerplate for a Discord bot. It includes a DiscordGo gateway client, a Fiber operational API with Scalar documentation, Redis and PostgreSQL adapters, deterministic clocks, reusable cron jobs, graceful shutdown, and a real-process E2E harness.

## Run

```sh
cp .env.example .env
go run ./cmd serve
```

`DISCORD_BOT_TOKEN` is mandatory. The process rejects startup before opening the HTTP server or Discord gateway when the variable is missing or empty.

The public endpoints are:

- `GET /status` for application and dependency status.
- `GET /docs` for the Scalar API reference.
- `GET /openapi.json` for the OpenAPI document.

Print the application version with:

```sh
go run ./cmd --version
```

## Structure

- `cmd/` contains the process entrypoint.
- `internal/cronjob/` contains the reusable asynchronous scheduler.
- `platform/discord/` wraps the single DiscordGo session and exposes it for bot handlers.
- `platform/httpapi/` contains Fiber, Scalar, OpenAPI, and graceful HTTP shutdown.
- `platform/redis/` and `platform/postgres/` contain reusable infrastructure clients.
- `platform/clock/` provides real and deterministic clocks.
- `platform/bootstrap/` owns dependency wiring and lifecycle.
- `e2e/` builds the real binary and validates the base wiring through HTTP.

## Validate

```sh
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
go test ./... -race
go build -trimpath -o /tmp/discord-bot ./cmd
docker build -t ghcr.io/pixelados-net/discord-bot:local .
```

This repository intentionally does not include Docker Compose or wiki content.
