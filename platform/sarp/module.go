package sarp

import (
	"github.com/pixelados-net/discord-bot/internal/performance"
	"go.uber.org/fx"
)

// Module provides the SARP business snapshot source.
var Module = fx.Module("sarp", fx.Provide(
	fx.Annotate(NewClient, fx.As(new(performance.Source))),
))
