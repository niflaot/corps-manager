ARG GO_IMAGE=golang:1.26.1-bookworm
FROM ${GO_IMAGE} AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY platform ./platform

ARG VERSION=2.0.0
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/discord-bot ./cmd

FROM scratch

ENV DISCORD_BOT_HOST=0.0.0.0 \
    DISCORD_BOT_ENVIRONMENT=production

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/discord-bot /discord-bot
EXPOSE 3100
ENTRYPOINT ["/discord-bot"]
