# discord-bot v1.2.0

Bot de Discord en Go para mantener mensajes editables y publicar el rendimiento de un negocio de SARP. Usa DiscordGo, Fiber, Uber Fx, Zap, PostgreSQL y Liquibase.

El dashboard es un mensaje administrado con Components V2. Su definición y el ID remoto quedan en PostgreSQL: cada actualización edita el mismo mensaje y el reconciliador lo restaura si fue alterado o eliminado.

## Configuración

```sh
cp .env.example .env
```

Variables obligatorias del proceso:

```dotenv
DISCORD_BOT_TOKEN=
DISCORD_BOT_GUILD_ID=
DISCORD_BOT_API_KEY=
```

El bot sólo necesita `View Channel`, `Send Messages` y `Manage Messages` en el canal configurado. No usa verificación, OAuth, Redis, el intent privilegiado de miembros ni requiere permiso de administrador.

Activa el monitor con:

```dotenv
DISCORD_BOT_PERFORMANCE_ENABLED=true
DISCORD_BOT_PERFORMANCE_ENDPOINT=https://flare.niflaot.dev/api/query
DISCORD_BOT_PERFORMANCE_ENDPOINT_TOKEN=
DISCORD_BOT_PERFORMANCE_BUSINESS_ID=1995
DISCORD_BOT_PERFORMANCE_CHANNEL_ID=123456789012345678
DISCORD_BOT_PERFORMANCE_INTERVAL=6h
DISCORD_BOT_PERFORMANCE_CUTOFF_WEEKDAY=Tuesday
DISCORD_BOT_PERFORMANCE_TIMEZONE=America/Bogota
```

`DISCORD_BOT_PERFORMANCE_ENDPOINT` debe apuntar al `POST /api/query` de `sarp-scrapper`. El bot envía esta consulta, por lo que el token real de SARP permanece únicamente en `sarp-scrapper`:

```json
{"provider":"gta-rol","path":"/businesses/1995"}
```

`DISCORD_BOT_PERFORMANCE_ENDPOINT_TOKEN` es opcional y sirve sólo si el túnel o reverse proxy protege ese endpoint con Bearer. No es el token de la API de SARP.

## Cálculo de ganancias

Al iniciar, el bot consulta inmediatamente y luego repite cada seis horas. El primer muestreo registra los valores actuales como histórico y establece la línea base semanal en cero. Los siguientes muestreos suman únicamente incrementos positivos del contador `earnings`; una reducción o reinicio del contador cambia la línea base sin restar del histórico.

Cada martes a las 00:00 en `America/Bogota` comienza un periodo nuevo. El corte anterior queda almacenado en PostgreSQL, incluyendo el total y el desglose por personaje. Se conservan 104 cortes por defecto. Los empleados que ya no aparecen quedan en el histórico, pero se marcan como inactivos.

## Base de datos

Ejecuta Liquibase antes del servicio:

```sh
cp database/liquibase.example.properties database/liquibase.properties
liquibase --defaults-file=database/liquibase.properties validate
liquibase --defaults-file=database/liquibase.properties update
```

Los changelogs de dominio están en `internal/messages/postgres/` e `internal/performance/postgres/`. La aplicación no modifica el esquema durante el arranque.

## Ejecutar

```sh
go run ./cmd serve
```

Rutas públicas:

- `GET /status`
- `GET /docs` y `GET /openapi.json` únicamente en `development`

Rutas protegidas con `Authorization: Bearer <DISCORD_BOT_API_KEY>`:

- `GET /api/performance`: estado persistido, totales, empleados y cortes.
- `POST /api/performance/refresh`: fuerza una consulta y actualiza el dashboard.
- `/api/messages`: administración genérica de mensajes editables.

Ejemplo de actualización manual:

```sh
curl -X POST http://127.0.0.1:3100/api/performance/refresh \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY"
```

## Estructura

- `internal/messages/`: mensajes durables y reconciliación con Discord.
- `internal/performance/`: periodos, deltas, histórico y dashboard.
- `platform/sarp/`: cliente del endpoint configurable de `sarp-scrapper`.
- `platform/httpapi/`: API Fiber y contrato OpenAPI/Scalar.
- `platform/discord/`: sesión DiscordGo y gateway de mensajes.
- `platform/postgres/`: pool PostgreSQL compartido.
- `internal/cronjob/`: trabajos periódicos context-aware.

## Validar

```sh
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
staticcheck ./...
go test ./... -race
go build -trimpath -o /tmp/discord-bot ./cmd
```

El stack para Portainer está en `deploy/portainer/mt-discord-bot/docker-compose.yml` y ejecuta la imagen de migraciones antes de iniciar el bot.
