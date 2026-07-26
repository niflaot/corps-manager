package localization

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contains localization catalog settings.
type Config struct {
	// Path optionally loads the catalog from a local file.
	Path string `env:"DISCORD_BOT_LOCALES_PATH" envDefault:""`
	// URL optionally loads the catalog from an HTTP or HTTPS endpoint.
	URL string `env:"DISCORD_BOT_LOCALES_URL" envDefault:""`
	// HTTPTimeout limits remote catalog loading during startup.
	HTTPTimeout time.Duration `env:"DISCORD_BOT_LOCALES_HTTP_TIMEOUT" envDefault:"10s"`
	// MaxBytes bounds local and remote catalog sizes.
	MaxBytes int64 `env:"DISCORD_BOT_LOCALES_MAX_BYTES" envDefault:"1048576"`
}

// LoadConfig reads localization configuration.
func LoadConfig() (Config, error) { return env.ParseAs[Config]() }
