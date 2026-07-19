package discordlinks

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contains durable OAuth artifact validity windows.
type Config struct {
	// IntentTTL controls how long a link attempt may wait to start.
	IntentTTL time.Duration `env:"DISCORD_BOT_OAUTH_INTENT_TTL" envDefault:"10m"`
	// ResultTTL controls how long a caller may exchange a completed result.
	ResultTTL time.Duration `env:"DISCORD_BOT_OAUTH_RESULT_TTL" envDefault:"5m"`
	// ArtifactRetention controls how long expired OAuth attempts remain for idempotent audit.
	ArtifactRetention time.Duration `env:"DISCORD_BOT_OAUTH_ARTIFACT_RETENTION" envDefault:"168h"`
}

// LoadConfig reads and validates Discord link domain configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	if config.IntentTTL <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_INTENT_TTL must be positive")
	}
	if config.ResultTTL <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_RESULT_TTL must be positive")
	}
	if config.ArtifactRetention < config.ResultTTL {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_ARTIFACT_RETENTION must not be shorter than result TTL")
	}
	return config, nil
}
