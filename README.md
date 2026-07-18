# discord-bot v1.0.0

[![Version](https://img.shields.io/badge/version-v1.0.0-5865F2.svg)](https://github.com/pixelados-net/discord-bot/releases/tag/v1.0.0)
[![CI](https://github.com/pixelados-net/discord-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/pixelados-net/discord-bot/actions/workflows/ci.yml)
[![Package](https://github.com/pixelados-net/discord-bot/actions/workflows/package.yml/badge.svg)](https://github.com/pixelados-net/discord-bot/actions/workflows/package.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/pixelados-net/discord-bot.svg)](https://pkg.go.dev/github.com/pixelados-net/discord-bot)

`discord-bot` is a production-oriented Go boilerplate for one Discord bot. It includes DiscordGo, an injected local event bus, a Fiber API with Scalar documentation, Redis and PostgreSQL adapters, Liquibase migrations, deterministic clocks, reusable cron jobs, graceful shutdown, and a real-process E2E harness.

Its first domain item is `messages`: PostgreSQL-backed static text/embed definitions assigned to Discord channels. A bounded reconciler checks integrity every minute, restores edited messages, and safely recreates missing messages with Discord's enforced nonce support.

## Run

```sh
cp .env.example .env
go run ./cmd serve
```

`DISCORD_BOT_TOKEN` and `DISCORD_BOT_API_KEY` are mandatory. The process rejects startup before opening the HTTP server or Discord gateway when either is invalid.

The public endpoints are:

- `GET /status` for application and dependency status.
- `GET /docs` for the Scalar API reference.
- `GET /openapi.json` for the OpenAPI document.

Every `/api/messages` route requires `Authorization: Bearer <DISCORD_BOT_API_KEY>`. Mutations also use `Idempotency-Key`; replacements and archival require the current numeric revision in `If-Match`. The full CRUD, assignment, reconciliation, schemas, and response contract is available at `/docs`.

## Database

PostgreSQL migrations use the same composed Liquibase layout as `pixels`. The root changelog is `database/changelog.xml`, while `messages` owns its changelog and versioned SQL under `internal/messages/postgres/`.

```sh
cp database/liquibase.example.properties database/liquibase.properties
liquibase --defaults-file=database/liquibase.properties validate
liquibase --defaults-file=database/liquibase.properties update
```

Liquibase must run before the bot starts. The application never mutates schema at runtime.

Create a managed message with:

```sh
curl -X POST http://127.0.0.1:3100/api/messages \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H "Idempotency-Key: rules-v1" \
  -H "Content-Type: application/json" \
  -d '{"key":"rules","guildId":"123456789012345678","channelId":"234567890123456789","payload":{"content":"Server rules","embeds":[],"allowedMentions":{"parse":[]}}}'
```

API writes are asynchronous with respect to Discord. Inspect `state`, `desiredHash`, `observedHash`, and `lastError`, or call `POST /api/messages/{key}/reconcile` to schedule an immediate check. Channel reassignments and archival deliberately leave the previous remote message untouched in v1.

## Local events

`platform/events` wraps `github.com/mariuswilms/bus` as a context-bound, injected event bus. Event names use lowercase dot-separated segments. Subscriptions are asynchronous and exact-name by default:

```go
unsubscribe, err := eventBus.Subscribe(ctx, "messages.reconciled", func(ctx context.Context, event events.Event) error {
	messageKey := event.Payload.(string)
	return handleReconciledMessage(ctx, messageKey)
})
defer unsubscribe()

_, err = eventBus.Publish("messages.reconciled", "rules")
```

The local bus is best-effort and scoped to one process. Use PostgreSQL or Redis when delivery must be durable or shared across replicas.

Print the application version with:

```sh
go run ./cmd --version
```

## Structure

- `cmd/` contains the process entrypoint.
- `internal/messages/` contains static-message domain logic and its PostgreSQL adapter.
- `internal/cronjob/` contains the reusable asynchronous scheduler.
- `platform/discord/` wraps the single DiscordGo session and exposes it for bot handlers.
- `platform/events/` wraps the process-local event bus.
- `platform/httpapi/` contains Fiber, Scalar, OpenAPI, and graceful HTTP shutdown.
- `platform/redis/` and `platform/postgres/` contain reusable infrastructure clients.
- `platform/clock/` provides real and deterministic clocks.
- `platform/bootstrap/` owns dependency wiring and lifecycle.
- `e2e/` builds the real binary and validates the base wiring through HTTP.
- `database/` contains the root Liquibase changelog and configuration templates.

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
