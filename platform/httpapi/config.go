package httpapi

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config contains HTTP API security and request limits.
type Config struct {
	// APIKey authenticates private API routes.
	APIKey string `env:"DISCORD_BOT_API_KEY"`
	// BodyLimit is the maximum accepted request body size.
	BodyLimit int `env:"DISCORD_BOT_HTTP_BODY_LIMIT" envDefault:"1048576"`
}

// LoadConfig reads and validates HTTP API configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.APIKey = strings.TrimSpace(config.APIKey)
	if config.APIKey == "" {
		return Config{}, fmt.Errorf("DISCORD_BOT_API_KEY is required")
	}
	if config.BodyLimit <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_HTTP_BODY_LIMIT must be positive")
	}
	return config, nil
}
