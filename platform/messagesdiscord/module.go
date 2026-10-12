package messagesdiscord

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/platform/clock"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// attachmentTimeout bounds the download of one JSON attachment.
const attachmentTimeout = 10 * time.Second

// attachmentHosts lists the only hosts attachments may be downloaded from.
var attachmentHosts = map[string]bool{"cdn.discordapp.com": true, "media.discordapp.net": true}

// Manager is the managed-message application surface used by the commands.
type Manager interface {
	// Create persists a new idempotent definition.
	Create(context.Context, messages.Definition, string) (messages.MutationResult, error)
	// Replace atomically replaces a revision.
	Replace(context.Context, string, uint64, messages.Definition, string) (messages.MutationResult, error)
	// Archive stops managing one message.
	Archive(context.Context, string, uint64, string) (messages.MutationResult, error)
	// Get reads one managed message.
	Get(context.Context, string) (messages.Record, error)
	// List returns a bounded page of managed messages.
	List(context.Context, messages.ListQuery) (messages.Page, error)
}

// Fetcher downloads a bounded attachment body.
type Fetcher func(ctx context.Context, rawURL string) ([]byte, error)

// Module contributes the /mensaje slash command.
var Module = fx.Module("messages-discord", fx.Provide(
	fx.Annotate(provideCommand, fx.ResultTags(`group:"discord_commands"`)),
))

func provideCommand(client *discord.Client, service *messages.Service, source clock.Clock, log *zap.Logger) discord.Command {
	c := &command{manager: service, guildID: client.GuildID(), fetch: httpFetcher(&http.Client{Timeout: attachmentTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}),
		pending: newPendingStore(source), log: log}
	return discord.Command{Definition: c.definition(), ComponentPrefix: customPrefix, Handle: c.handle}
}

func httpFetcher(client *http.Client) Fetcher {
	return func(ctx context.Context, rawURL string) ([]byte, error) {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || !attachmentHosts[parsed.Host] {
			return nil, fmt.Errorf("origen de archivo no permitido")
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("respuesta inesperada de Discord: %d", response.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxPayloadBytes+1))
		if err != nil {
			return nil, err
		}
		return body, nil
	}
}
