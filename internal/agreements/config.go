package agreements

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// DefaultRefreshInterval is the default six-hour agreement panel refresh.
const DefaultRefreshInterval = 6 * time.Hour

var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Config controls the agreements list and control panel.
type Config struct {
	// Enabled activates agreement interactions and publishing; disabled by default.
	Enabled bool `env:"DISCORD_BOT_AGREEMENTS_ENABLED"`
	// ControlChannelID selects the channel containing the add button.
	ControlChannelID string `env:"DISCORD_BOT_AGREEMENTS_CONTROL_CHANNEL_ID"`
	// RefreshInterval defaults to six hours between panel refreshes.
	RefreshInterval time.Duration `env:"DISCORD_BOT_AGREEMENTS_REFRESH_INTERVAL"`
}

// LoadConfig reads and validates agreement configuration.
func LoadConfig() (Config, error) {
	config := Config{RefreshInterval: DefaultRefreshInterval}
	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}
	config.ControlChannelID = strings.TrimSpace(config.ControlChannelID)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_AGREEMENTS_REFRESH_INTERVAL must be positive")
	}
	if config.Enabled && !snowflakePattern.MatchString(config.ControlChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_AGREEMENTS_CONTROL_CHANNEL_ID must be a Discord snowflake")
	}
	return config, nil
}
