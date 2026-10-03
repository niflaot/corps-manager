package securitydiscord

import (
	"context"

	"github.com/niflaot/corps-manager/internal/security"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module exposes the Discord security gateway and lifecycle-bound handlers.
var Module = fx.Module("security-discord", fx.Provide(fx.Annotate(NewGateway, fx.As(new(security.Gateway)))), fx.Invoke(register))

func register(lifecycle fx.Lifecycle, client *discord.Client, service *security.Service, log *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	handler := &handler{ctx: ctx, client: client, service: service, log: log}
	removeInteraction := client.AddHandler(handler.interaction)
	removeMessage := client.AddHandler(handler.message)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		cancel()
		removeInteraction()
		removeMessage()
		return nil
	}})
}
