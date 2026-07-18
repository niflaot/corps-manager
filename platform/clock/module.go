package clock

import "go.uber.org/fx"

// Module provides the production clock implementation.
var Module = fx.Module("clock", fx.Provide(
	fx.Annotate(provideClock, fx.As(new(Clock))),
))

func provideClock() Real { return Real{} }
