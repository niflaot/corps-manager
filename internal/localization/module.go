package localization

import (
	"github.com/caarlos0/env/v11"
	"go.uber.org/fx"
)

// Config contains localization catalog settings.
type Config struct {
	// Path optionally overrides the embedded localization catalog.
	Path string `env:"DISCORD_BOT_LOCALES_PATH"`
}

// Module provides localization configuration and the immutable catalog.
var Module = fx.Module("localization", fx.Provide(LoadConfig, provideCatalog))

// LoadConfig reads localization configuration.
func LoadConfig() (Config, error) { return env.ParseAs[Config]() }

func provideCatalog(config Config) (*Catalog, error) { return Load(config.Path) }
