# AGENTS

- Keep the repository a reusable Discord bot boilerplate; do not add unrelated product or storage concerns.
- All code must be performant, asynchronous where I/O is involved, and safe to stop through `context.Context`.
- All exported functions, types, constants, variables, struct fields, and interfaces must use Go doc style comments.
- Do not add comments inside code unless they clarify non-obvious behavior.
- Keep every code file under 250 lines and every package at a maximum of 6 source/test file pairs.
- Keep domain jobs under `internal/`; reusable runtime adapters belong under `platform/`.
- Preserve dependency injection for Discord, local events, Redis, PostgreSQL, clocks, cron jobs, and HTTP health checks.
- Manage PostgreSQL schema changes through `database/changelog.xml` and domain-owned Liquibase changelogs; do not add standalone migration SQL.
- Follow standard Go style and avoid unnecessary abstractions or dependencies.
- Do not add Docker Compose files or wiki documentation.
- Run `gofmt`, `go test ./...`, `go vet ./...`, and `go test ./... -race` before handing off changes.
