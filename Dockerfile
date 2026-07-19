FROM golang:1.26-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY locales ./locales
COPY platform ./platform

ARG VERSION=1.1.0
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/discord-bot ./cmd

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/discord-bot /discord-bot
EXPOSE 3100
ENTRYPOINT ["/discord-bot"]
