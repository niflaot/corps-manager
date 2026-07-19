# discord-bot v1.1.1

[![Version](https://img.shields.io/badge/version-v1.1.1-5865F2.svg)](https://github.com/pixelados-net/discord-bot/releases/tag/v1.1.1)
[![CI](https://github.com/pixelados-net/discord-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/pixelados-net/discord-bot/actions/workflows/ci.yml)
[![Package](https://github.com/pixelados-net/discord-bot/actions/workflows/package.yml/badge.svg)](https://github.com/pixelados-net/discord-bot/actions/workflows/package.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/pixelados-net/discord-bot.svg)](https://pkg.go.dev/github.com/pixelados-net/discord-bot)

`discord-bot` is a production-oriented Go boilerplate for one Discord bot. It includes DiscordGo, an injected local event bus, a Fiber API with Scalar documentation, Redis and PostgreSQL adapters, Liquibase migrations, deterministic clocks, reusable cron jobs, graceful shutdown, and a real-process E2E harness.

Its first domain item is `messages`: PostgreSQL-backed Discord Components V2 definitions assigned to channels. A bounded reconciler checks integrity every minute, restores edited messages, and safely recreates missing messages with Discord's enforced nonce support. The verification guard adds SQL-backed settings, up to five role groups, timestamped multi-memberships, localized DMs with unverify buttons, read-only verification markup, and an automatically repaired anti-bot trap channel.

## Run

```sh
cp .env.example .env
go run ./cmd serve
```

`DISCORD_BOT_TOKEN`, `DISCORD_BOT_GUILD_ID`, and `DISCORD_BOT_API_KEY` are mandatory. Before opening Fiber or the gateway, the process authenticates through Discord REST, requires the bot to belong exclusively to that guild, and requires `ADMINISTRATOR` there. Enable **Server Members Intent** under the application's Bot settings in the Discord Developer Portal; Discord requires this privileged intent for member join and removal events. `DISCORD_BOT_LOCALES_PATH` optionally points to a replacement JSON catalog; the embedded [`locales/messages.json`](locales/messages.json) is the default.

Logging is configured independently from the environment:

```dotenv
DISCORD_BOT_LOG_LEVEL=info
DISCORD_BOT_LOG_FORMAT=console
```

`DISCORD_BOT_LOG_LEVEL` accepts Zap levels such as `debug`, `info`, `warn`, and `error`. `DISCORD_BOT_LOG_FORMAT` accepts `console` for local readability or `json` for structured ingestion.
Fiber requests and DiscordGo internals share this same injected Zap logger. Successful recurring dependency health snapshots are emitted only at `debug`; unavailable dependencies are emitted at `error`.

The public endpoint is:

- `GET /status` for application and dependency status.

Only when `DISCORD_BOT_ENVIRONMENT=development`, `GET /docs` exposes the Scalar API reference and `GET /openapi.json` exposes its OpenAPI document. Both routes return `404` in `test` and `production`.

Every `/api` route requires `Authorization: Bearer <DISCORD_BOT_API_KEY>`. Message mutations also use `Idempotency-Key`; replacements and archival require the current numeric revision in `If-Match`. Settings use dotted keys and support `GET`, revision-aware `PUT`, and `DELETE` reset. Verification groups, memberships, and manual reconciliation live under `/api/verification`. The full contract is available at `/docs`.

## Discord identity links

The optional identity-link service owns Discord OAuth from authorization through token revocation. A caller owns its local sessions and supplies only a stable opaque `subject`; the bot stores the authoritative association between that subject and Discord's immutable user ID. OAuth user tokens are used only to read `/users/@me`, then revoked and discarded. Bot-token actions remain independent from user OAuth.

Enable the service with:

```dotenv
DISCORD_BOT_OAUTH_ENABLED=true
DISCORD_BOT_OAUTH_CLIENT_ID=123456789012345678
DISCORD_BOT_OAUTH_CLIENT_SECRET=replace-me
DISCORD_BOT_OAUTH_PUBLIC_URL=http://localhost:3100
DISCORD_BOT_OAUTH_COMPLETION_URLS={"pixelados-links":"http://localhost:3000/configuracion/vinculos/discord/resultado","pixelados-login":"http://localhost:3000/auth/discord/resultado"}
```

Register this exact redirect URI in the Discord Developer Portal:

```text
http://localhost:3100/oauth/discord/callback
```

Production public and completion URLs must use HTTPS. Completion URLs are a startup-time allowlist keyed by `completionKey`; request bodies cannot introduce arbitrary return URLs. Only these two browser routes are public:

- `GET /oauth/discord/start/{intentId}`
- `GET /oauth/discord/callback`

Intent creation, result exchange, link inspection, and unlinking remain under `/api/discord-links` and require the configured Bearer API key. Link and result creation also require `Idempotency-Key`.

### Link a test user

The caller assigns the local identifier. Use the immutable account ID from the caller, namespaced to avoid ambiguity; for example `user:test-001`. The service generates the link's separate internal UUID after successful OAuth.

```sh
curl -X POST http://127.0.0.1:3100/api/discord-links/intents \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H "Idempotency-Key: link-user-test-001" \
  -H "Content-Type: application/json" \
  -d '{"subject":"user:test-001","completionKey":"pixelados-links"}'
```

Open the returned `startUrl` in the user's browser. Discord returns to the bot, and the bot redirects to the configured Pixelados completion URL with a short-lived `code`. Pixelados must exchange that code from its backend, using the same idempotency key for every retry:

```sh
curl -X POST http://127.0.0.1:3100/api/discord-links/results/exchange \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H "Idempotency-Key: exchange-link-user-test-001" \
  -H "Content-Type: application/json" \
  -d '{"code":"CODE_FROM_COMPLETION_REDIRECT"}'
```

A successful result has `status: "linked"`, the original `subject`, and a `link` containing its generated `id`, Discord user ID, profile snapshot, granted scopes, status, and timestamps. A different idempotency key cannot consume the same result code again.

Inspect the latest link history by caller subject:

```sh
curl http://127.0.0.1:3100/api/discord-links/subjects/user:test-001 \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY"
```

Inspect the active owner of a Discord user ID:

```sh
curl http://127.0.0.1:3100/api/discord-links/users/123456789012345678 \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY"
```

Unlinking is naturally idempotent by stable subject and preserves historical timestamps:

```sh
curl -X DELETE http://127.0.0.1:3100/api/discord-links/subjects/user:test-001 \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY"
```

### Authenticate through an existing link

Create a login intent without a subject. This prevents the browser from choosing which local account it wants to enter:

```sh
curl -X POST http://127.0.0.1:3100/api/discord-links/login-intents \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H "Idempotency-Key: login-attempt-001" \
  -H "Content-Type: application/json" \
  -d '{"completionKey":"pixelados-login"}'
```

After the same browser OAuth sequence, exchange the returned completion code. An active association returns `status: "authenticated"`, its trusted `subject`, and the current link snapshot. An unknown Discord identity returns `status: "not_linked"` and no subject. Pixelados must still validate the local account and create its own session; the bot never creates or receives Pixelados cookies.

Possible result statuses are `linked`, `authenticated`, `not_linked`, `denied`, `conflict`, and `failed`. Safe `errorCode` values distinguish authorization denial, provider errors, occupied subjects, occupied Discord identities, and users that are not linked. The full request and response schemas are in `/docs` when running in development.

Expired intents and result credentials are retained for seven days by default to preserve idempotent retries and short operational audit, then removed by the context-bound cron scheduler. Change this with `DISCORD_BOT_OAUTH_ARTIFACT_RETENTION`; durable link and unlink history is never removed by that cleanup.

## Database

PostgreSQL migrations use the same composed Liquibase layout as `pixels`. The root changelog is `database/changelog.xml`, while `messages` owns its changelog and versioned SQL under `internal/messages/postgres/`.

```sh
cp database/liquibase.example.properties database/liquibase.properties
liquibase --defaults-file=database/liquibase.properties validate
liquibase --defaults-file=database/liquibase.properties update
```

Liquibase must run before the bot starts. The application never mutates schema at runtime.

Tagged releases also publish
`ghcr.io/pixelados-net/discord-bot-migrations:<version>`. That image contains
the changelog from the same commit as the application image and uses Liquibase
4.31 as its entrypoint. Deployment orchestration should run its `update`
command to successful completion before starting the matching bot version.

Create a managed message with:

```sh
curl -X POST http://127.0.0.1:3100/api/messages \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H "Idempotency-Key: rules-v1" \
  -H "Content-Type: application/json" \
  -d '{"key":"verification","guildId":"123456789012345678","channelId":"234567890123456789","payload":{"components":[{"type":10,"content":"Choose your verification groups"}],"allowedMentions":{"parse":[]}}}'
```

Select that managed message and create a verification group:

```sh
curl -X PUT http://127.0.0.1:3100/api/settings/verification.message.key \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" -H "Content-Type: application/json" \
  -d '{"value":"verification"}'

curl -X POST http://127.0.0.1:3100/api/verification/groups \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" -H "Content-Type: application/json" \
  -d '{"key":"member","roleId":"345678901234567890","buttonLabel":"Member","buttonEmoji":"✅","buttonStyle":3,"position":1,"enabled":true}'

curl -X POST http://127.0.0.1:3100/api/verification/reconcile \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY"
```

The target role must exist, be non-managed, and remain below the bot's highest role. Users may hold several verification memberships simultaneously. Repeating verification is idempotent; unverify removes only the selected role and hard-deletes that membership. Leaving the guild invalidates every membership. Join/remove events reconcile immediately, while the periodic audit compares Discord's current `joined_at` with each `verified_at`, deletes stale records missed during downtime, and restores missing roles for current records. Discord privacy settings can prevent a DM, but never roll back a successful role and membership.

API message writes are asynchronous with respect to Discord. Inspect `state`, `desiredHash`, `observedHash`, and `lastError`, or schedule immediate reconciliation. Components V2 disables legacy `content` and `embeds`; the API validates Discord component nesting, button constraints, and the 40-component message limit before persistence.

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

## Release

GHCR publication runs only for tags matching `v*.*.*`. The release workflow validates tests, Vet, Staticcheck and compilation, builds all supported binary targets, and then publishes matching multi-architecture application and migration images. A tag such as `v1.2.3` produces `v1.2.3`, `1.2.3`, `1.2`, `1`, and `latest` tags on both image repositories and embeds `1.2.3` in the application binary.

Configure the repository Actions secret `DISCORD_WEBHOOK_URL` to enable the tag notification workflow, then publish a release with:

```sh
git tag v1.2.3
git push origin v1.2.3
```

## Structure

- `cmd/` contains the process entrypoint.
- `internal/messages/` contains static-message domain logic and its PostgreSQL adapter.
- `internal/settings/` contains typed dotted settings and code defaults.
- `internal/verification/` contains role groups, memberships, and the guard reconciler.
- `internal/discordlinks/` contains OAuth intent, result, login, and durable link behavior.
- `internal/cronjob/` contains the reusable asynchronous scheduler.
- `platform/discord/` wraps the single DiscordGo session and exposes it for bot handlers.
- `platform/discordoauth/` owns the bounded confidential OAuth HTTP client.
- `platform/events/` wraps the process-local event bus.
- `platform/httpapi/` contains Fiber, Scalar, OpenAPI, and graceful HTTP shutdown.
- `platform/redis/` and `platform/postgres/` contain reusable infrastructure clients.
- `platform/clock/` provides real and deterministic clocks.
- `platform/bootstrap/` composes focused Uber Fx modules and owns concurrent runtime startup.
- Every DI-enabled domain and adapter owns its `module.go`; bootstrap only composes those modules and owns the concurrent runtime.
- `e2e/` builds the real binary and validates the base wiring through HTTP.
- `database/` contains the root Liquibase changelog and configuration templates.

## Validate

```sh
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
staticcheck ./...
go test ./... -race
go build -trimpath -o /tmp/discord-bot ./cmd
docker build -t ghcr.io/pixelados-net/discord-bot:local .
```

This repository intentionally does not include Docker Compose or wiki content.
