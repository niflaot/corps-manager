package discord

import (
	"context"

	"github.com/pixelados-net/discord-bot/internal/localization"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/internal/verification"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Handlers owns registered Discord event handler removers.
type Handlers struct{ removers []func() }

// Module provides the single-guild client, gateways, handlers, and startup preflight.
var Module = fx.Module("discord", fx.Provide(
	LoadConfig,
	provideClient,
	fx.Annotate(provideGuildID, fx.ResultTags(`name:"guild_id"`)),
	fx.Annotate(provideMessageGateway, fx.As(new(messages.Gateway))),
	fx.Annotate(provideVerificationGateway, fx.As(new(verification.Gateway))),
	fx.Annotate(provideGuardGateway, fx.As(new(verification.GuardGateway)), fx.As(fx.Self())),
	provideGuildGateway,
	provideHandlers,
))

func provideClient(lifecycle fx.Lifecycle, config Config, log *zap.Logger) (*Client, error) {
	client, err := New(config, log)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return client.ValidateGuildAdministrator(ctx)
	}})
	return client, nil
}

func provideGuildID(config Config) string { return config.GuildID }

func provideMessageGateway(client *Client, config Config) *MessageGateway {
	return NewMessageGateway(client, config.GuildID)
}

func provideVerificationGateway(client *Client) *VerificationGateway {
	return NewVerificationGateway(client)
}

func provideGuardGateway(client *Client) *GuardGateway { return NewGuardGateway(client) }

func provideGuildGateway(client *Client) *GuildGateway { return NewGuildGateway(client) }

func provideHandlers(lifecycle fx.Lifecycle, client *Client, service *verification.Service,
	catalog *localization.Catalog, gateway *GuardGateway) Handlers {
	handlers := Handlers{removers: []func(){
		RegisterVerificationHandlers(client, service, catalog), gateway.RegisterTrapHandler(),
	}}
	handlers.removers = append(handlers.removers, RegisterMemberLifecycleHandlers(client, service)...)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		for _, remove := range handlers.removers {
			remove()
		}
		return nil
	}})
	return handlers
}
