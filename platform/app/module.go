package app

import "go.uber.org/fx"

// Version is the injected application build version.
type Version string

// Module provides application-level runtime configuration.
var Module = fx.Module("app", fx.Provide(LoadConfig))
