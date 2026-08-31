package agreements

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Config controls the agreements list and control panel.
type Config struct {
	// Enabled activates agreement interactions and publishing.
	Enabled bool `env:"DISCORD_BOT_AGREEMENTS_ENABLED" envDefault:"true"`
	// ChannelID selects the public agreements-list channel.
	ChannelID string `env:"DISCORD_BOT_AGREEMENTS_CHANNEL_ID"`
	// ControlChannelID selects the channel containing the add button.
	ControlChannelID string `env:"DISCORD_BOT_PERFORMANCE_CHANNEL_ID"`
	// RefreshInterval controls periodic panel reconciliation.
	RefreshInterval time.Duration `env:"DISCORD_BOT_AGREEMENTS_REFRESH_INTERVAL" envDefault:"6h"`
}

// LoadConfig reads and validates agreement configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	config.ControlChannelID = strings.TrimSpace(config.ControlChannelID)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_AGREEMENTS_REFRESH_INTERVAL must be positive")
	}
	if config.Enabled && !snowflakePattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_AGREEMENTS_CHANNEL_ID must be a Discord snowflake")
	}
	if config.Enabled && !snowflakePattern.MatchString(config.ControlChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_PERFORMANCE_CHANNEL_ID must be a Discord snowflake")
	}
	return config, nil
}
