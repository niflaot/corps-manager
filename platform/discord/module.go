package discord

import (
	"context"

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
	fx.Annotate(provideRouter, fx.ParamTags("", "", `group:"discord_commands"`, "")),
), fx.Invoke(registerCommands))

func provideRouter(lifecycle fx.Lifecycle, client *Client, commands []Command, log *zap.Logger) *Router {
	ctx, cancel := context.WithCancel(context.Background())
	router := NewRouter(ctx, client.GuildID(), commands, log)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		cancel()
		return nil
	}})
	return router
}

func registerCommands(lifecycle fx.Lifecycle, client *Client, router *Router) {
	if len(router.commands) == 0 {
		return
	}
	remove := client.AddHandler(router.Handle)
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error { return router.Register(ctx, client) },
		OnStop:  func(context.Context) error { remove(); return nil },
	})
}

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
