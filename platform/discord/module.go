package discord

import (
	"context"

	"github.com/niflaot/corps-manager/internal/announcements"
	"github.com/niflaot/corps-manager/internal/messages"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the single-guild client and managed-message gateway.
var Module = fx.Module("discord", fx.Provide(
	LoadConfig,
	provideClient,
	fx.Annotate(provideGuildID, fx.ResultTags(`name:"guild_id"`)),
	fx.Annotate(provideMessageGateway, fx.As(new(messages.Gateway))),
	fx.Annotate(provideOpeningGateway, fx.As(new(announcements.Gateway))),
), fx.Invoke(registerInactivityInteractions))

func provideClient(lifecycle fx.Lifecycle, config Config, log *zap.Logger) (*Client, error) {
	client, err := New(config, log)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return client.ValidateAuthentication(ctx)
	}})
	return client, nil
}

func provideGuildID(config Config) string { return config.GuildID }

func provideMessageGateway(client *Client, config Config) *MessageGateway {
	return NewMessageGateway(client, config.GuildID)
}

func provideOpeningGateway(client *Client) *OpeningGateway { return NewOpeningGateway(client) }
