package events

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the process-local event bus and lifecycle cleanup.
var Module = fx.Module("events", fx.Provide(provideBus))

func provideBus(lifecycle fx.Lifecycle, log *zap.Logger) *Bus {
	bus := New(context.Background(), log)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		bus.Close()
		return nil
	}})
	return bus
}
