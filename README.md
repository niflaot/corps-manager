# corps-manager v2.0.0

Bot modular de Discord en Go con mensajes administrados e idempotentes, verificación por rol, canal anti bots y convenios por empresa. Usa DiscordGo, Fiber, Uber Fx, Zap, PostgreSQL y Liquibase.

## Configuración y arranque

```sh
cp .env.example .env
cp database/liquibase.example.properties database/liquibase.properties
liquibase --defaults-file=database/liquibase.properties validate
liquibase --defaults-file=database/liquibase.properties update
go run ./cmd serve
```

Configura `DISCORD_BOT_TOKEN`, `DISCORD_BOT_GUILD_ID`, `DISCORD_BOT_API_KEY` y las variables PostgreSQL. El bot administra únicamente esa guild. Los controles nuevos están deshabilitados por defecto para permitir configurar sus recursos antes de activarlos.

La aplicación no aplica migraciones al iniciar. `go run ./cmd version` muestra la versión del binario.

## Mensajes administrados

La API mantiene la definición y el ID remoto en PostgreSQL. El reconciliador publica Components V2, edita el mismo mensaje y lo recrea si desaparece. Las mutaciones requieren `Idempotency-Key`; los reemplazos y archivados requieren además `If-Match` con la revisión actual. Repetir una operación con la misma clave devuelve su resultado anterior; reutilizarla con otro contenido produce conflicto. Las claves idempotentes se conservan 24 horas.

```sh
curl -X POST http://127.0.0.1:3100/api/messages \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: verification-create-1' \
  -d '{
    "key":"verification",
    "guildId":"123456789012345678",
    "channelId":"234567890123456789",
    "payload":{
      "components":[{"type":10,"content":"# Bienvenido\nLee las reglas y verifica tu acceso."}],
      "allowedMentions":{"parse":[]}
    }
  }'
```

## Verificación por rol

Crea tu mensaje mediante la API y configura su key:

```dotenv
DISCORD_BOT_VERIFICATION_ENABLED=true
DISCORD_BOT_VERIFICATION_CHANNEL_ID=234567890123456789
DISCORD_BOT_VERIFICATION_MESSAGE_KEY=verification
DISCORD_BOT_VERIFICATION_ROLE_ID=345678901234567890
DISCORD_BOT_SECURITY_REFRESH_INTERVAL=1m
```

El control añade un único botón **Verificarme** y conserva tu contenido. Cada pulsación válida asigna el rol al autor de la interacción. Solo se acepta el mensaje remoto actual, en el canal y la guild configurados; los mensajes copiados, antiguos o archivados no verifican. Asignar de nuevo el mismo rol es idempotente.

Puedes editar el contenido mediante la API; consulta primero la revisión actual porque la inserción del control también modifica la definición. Deja espacio para una fila y un botón dentro de los límites de Components V2. El control se instala al iniciar y se revisa periódicamente. Si la key aún no existe, se registra el error y se vuelve a intentar en la próxima revisión.

Esta versión verifica mediante una pulsación, sin CAPTCHA ni comprobación de identidad externa. El bot necesita `Manage Roles` y su rol debe estar por encima del rol que entrega.

## Canal anti bots

```dotenv
DISCORD_BOT_ANTIBOT_ENABLED=true
DISCORD_BOT_SECURITY_REFRESH_INTERVAL=1m
```

El bot crea o adopta un canal de texto llamado **no-escribir** dentro de la categoría **Control anti bots**, ubicada al final del servidor. Publica un mensaje administrado con el texto **no escribir, control anti bots**.

Cualquier autor que publique allí se banea, incluidos otros bots; únicamente se excluye este bot. No se borran mensajes históricos al banear. Discord puede impedir el baneo del propietario o de miembros con una jerarquía superior; estos fallos se registran como errores.

La revisión al iniciar y cada minuto, por defecto:

- Conserva el canal canónico mediante su asignación persistida; lo recrea si desaparece.
- Repara nombre, categoría, posición, descripción y permisos del canal.
- Mantiene el canal visible y escribible para `@everyone`, y desactiva la creación de hilos.
- Elimina otros canales llamados exactamente `no-escribir` en la guild configurada.
- Solicita la reparación del aviso administrado y elimina mensajes adicionales del propio bot cuando el aviso canónico existe.

El nombre `no-escribir` queda reservado para este control. El bot necesita `Manage Channels`, `Ban Members`, `View Channel`, `Send Messages`, `Read Message History` y `Manage Messages`. No necesita intent privilegiado de miembros ni de contenido de mensajes. Ejecuta una sola instancia activa del bot para coordinar la creación de canales.

## Empresas y convenios

Las empresas son un catálogo dinámico, sin tres tipos fijos. Se administran mediante la API protegida:

```sh
curl -X POST http://127.0.0.1:3100/api/companies \
  -H "Authorization: Bearer $DISCORD_BOT_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"id":"empresa-a","name":"Empresa A","channelId":"567890123456789012"}'
```

Cada empresa tiene su propio canal (`channelId`) donde el bot publica el listado de sus convenios; así cada empresa muestra solo los suyos. `PUT /api/companies/:id` actualiza nombre y canal. `GET /api/companies` lista las empresas. `DELETE /api/companies/:id` elimina una empresa vacía; devuelve `409` si tiene convenios. El identificador de empresa es único. La creación duplicada devuelve `409` y no genera otra empresa. Estas operaciones están disponibles aunque el panel de Discord esté deshabilitado.

```dotenv
DISCORD_BOT_AGREEMENTS_ENABLED=true
DISCORD_BOT_AGREEMENTS_CONTROL_CHANNEL_ID=456789012345678901
DISCORD_BOT_AGREEMENTS_REFRESH_INTERVAL=6h
```

El canal de control conserva el panel administrado con **Añadir convenio**, **Ver convenios**, **Añadir empresa**, **Editar empresa** y **Eliminar empresa**. Las empresas (identificador, nombre y canal de convenios) se configuran ahí mismo mediante formularios, sin variable de canal global; cada cambio republica el listado de la empresa en su canal. Una empresa sin canal (creada antes de esta versión) no se publica hasta editarla. Eliminar una empresa archiva su mensaje administrado y su identificador no puede reutilizarse para publicar. Al añadir, eliges una empresa en un selector privado paginado y completas el formulario con identificador, descripción e imagen HTTPS opcional. El catálogo admite más de 25 empresas mediante paginación.

El identificador del convenio es único **dentro de su empresa**: dos empresas pueden usar el mismo. La descripción admite de 3 a 1000 caracteres. El listado público muestra la empresa de cada convenio y hasta tres convenios completos para respetar los límites de Discord. La consulta privada permite recorrer todo el listado. Quienes tengan acceso al canal de control pueden registrar convenios; restringe ese canal al equipo que los administra.

## API

Rutas públicas: `GET /status`; `GET /docs` y `GET /openapi.json` solamente en `development`.

Todas las rutas `/api` requieren `Authorization: Bearer <DISCORD_BOT_API_KEY>`:

| Método | Ruta | Acción |
|---|---|---|
| GET / POST | `/api/messages` | Listar o crear mensajes |
| GET / PUT / DELETE | `/api/messages/:key` | Consultar, reemplazar o archivar |
| PUT | `/api/messages/:key/assignment` | Cambiar asignación |
| POST | `/api/messages/:key/reconcile` | Pedir revisión inmediata |
| GET / POST | `/api/companies` | Listar o crear empresas |
| PUT / DELETE | `/api/companies/:id` | Actualizar nombre y canal, o eliminar una empresa vacía |

## Actualización desde la versión anterior

La migración de v2 elimina las tablas de rendimiento SARP, inactividad, anuncios y clientes frecuentes. Vacía los convenios anteriores y añade empresas y la clave compuesta de convenios. No crea empresas iniciales automáticamente.

Los mensajes personalizados y su idempotencia se conservan. Las definiciones de los antiguos paneles se archivan; archivar detiene su reconciliación y no borra el mensaje ya publicado en Discord. Esos mensajes antiguos pueden retirarse manualmente. Los paneles de convenios se reinician con un aviso vacío y adoptan el nuevo contenido al habilitar el módulo.

Los directorios de rendimiento, inactividad y anuncios conservan únicamente el historial de Liquibase y sus cambios de retirada. Clientes se eliminó por completo de `internal/` y `platform/`; `database/retired-customers.xml` solo limpia sus tablas y archiva el panel en bases existentes, sin recrear su esquema en instalaciones nuevas.

Se retiraron también las variables `PERFORMANCE_*`, `INACTIVITY_*`, `ANNOUNCEMENT_*` y `CUSTOMERS_*`, el directorio web `/customers` y sus rutas API. El canal de control de convenios ahora usa `DISCORD_BOT_AGREEMENTS_CONTROL_CHANNEL_ID`.

## Arquitectura y despliegue

- `internal/messages`: persistencia, idempotencia y reconciliación de mensajes.
- `internal/security`: políticas de verificación y anti bots.
- `internal/agreements`: empresas, convenios y paneles.
- `internal/cronjob`: trabajos cancelables por contexto.
- `platform/`: adaptadores de Discord, API HTTP, PostgreSQL, reloj, logs y composición Fx.

El Dockerfile compila con `golang:1.26.1-bookworm` y produce una imagen mínima que escucha en `0.0.0.0:3100`; `Dockerfile.migrations` incluye todos los changelogs. Ambos excluyen credenciales locales del contexto de construcción. El stack existente de Portainer descarga la referencia Git seleccionada, aplica Liquibase y compila el bot con `DISCORD_BOT_GO_IMAGE`: este flujo no ejecuta el Dockerfile. GHCR solo se publica desde tags `v*.*.*`, tras validar y compilar la versión.

## Validación

```sh
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
go test ./... -race
go build -trimpath -o /tmp/discord-bot ./cmd
```

Las pruebas de integración de PostgreSQL requieren `DISCORD_BOT_INTEGRATION_POSTGRES_DSN` apuntando a una base de pruebas con Liquibase aplicado. Usan tablas de prueba truncadas: no apuntes esa variable a una base con datos que quieras conservar.
