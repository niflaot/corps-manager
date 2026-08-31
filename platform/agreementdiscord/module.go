// Package agreementdiscord handles business-agreement Discord interactions.
package agreementdiscord

import (
	"context"

	"github.com/niflaot/corps-manager/internal/agreements"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module registers the agreement Discord interaction adapter.
var Module = fx.Module("agreement-discord", fx.Invoke(register))

func register(lifecycle fx.Lifecycle, client *discord.Client, service *agreements.Service,
	config agreements.Config, log *zap.Logger) {
	handler := &handler{service: service, config: config, log: log}
	remove := client.AddHandler(handler.handle)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		remove()
		return nil
	}})
}
