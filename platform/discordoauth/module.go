package discordoauth

import (
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides optional Discord OAuth configuration and its HTTP adapter.
var Module = fx.Module("discordoauth", fx.Provide(
	LoadConfig,
	fx.Annotate(provideClient, fx.As(new(discordlinks.OAuthGateway)), fx.As(fx.Self())),
))

func provideClient(config Config, log *zap.Logger) *Client { return New(config, log) }
