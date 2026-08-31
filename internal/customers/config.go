package customers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Config controls the frequent-customer panel.
type Config struct {
	// Enabled activates customer interactions and publishing.
	Enabled bool `env:"DISCORD_BOT_CUSTOMERS_ENABLED" envDefault:"true"`
	// ChannelID selects the channel containing the customer panel.
	ChannelID string `env:"DISCORD_BOT_CUSTOMERS_CHANNEL_ID"`
	// RefreshInterval controls periodic panel reconciliation.
	RefreshInterval time.Duration `env:"DISCORD_BOT_CUSTOMERS_REFRESH_INTERVAL" envDefault:"6h"`
}

// LoadConfig reads and validates customer configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_CUSTOMERS_REFRESH_INTERVAL must be positive")
	}
	if config.Enabled && !snowflakePattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_CUSTOMERS_CHANNEL_ID must be a Discord snowflake")
	}
	return config, nil
}
