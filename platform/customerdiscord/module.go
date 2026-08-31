// Package customerdiscord handles frequent-customer Discord interactions.
package customerdiscord

import (
	"context"

	"github.com/niflaot/corps-manager/internal/customers"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module registers the frequent-customer Discord interaction adapter.
var Module = fx.Module("customer-discord", fx.Invoke(register))

func register(lifecycle fx.Lifecycle, client *discord.Client, service *customers.Service,
	config customers.Config, log *zap.Logger) {
	handler := &handler{service: service, config: config, log: log}
	remove := client.AddHandler(handler.handle)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		remove()
		return nil
	}})
}
