package inactivity

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var discordSnowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Config controls the inactivity dismissal registry and Discord dashboard.
type Config struct {
	// Enabled activates the registry dashboard and interactions.
	Enabled bool `env:"DISCORD_BOT_INACTIVITY_ENABLED" envDefault:"true"`
	// ChannelID selects the Discord channel containing the registry message.
	ChannelID string `env:"DISCORD_BOT_PERFORMANCE_CHANNEL_ID"`
	// AnnouncementChannelID receives public business-opening announcements.
	AnnouncementChannelID string `env:"DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID"`
	// RefreshInterval controls periodic dashboard reconciliation.
	RefreshInterval time.Duration `env:"DISCORD_BOT_INACTIVITY_REFRESH_INTERVAL" envDefault:"6h"`
}

// LoadConfig reads and validates inactivity registry configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	config.AnnouncementChannelID = strings.TrimSpace(config.AnnouncementChannelID)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_INACTIVITY_REFRESH_INTERVAL must be positive")
	}
	if !config.Enabled {
		return config, nil
	}
	if !discordSnowflakePattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_PERFORMANCE_CHANNEL_ID must be a Discord snowflake")
	}
	if !discordSnowflakePattern.MatchString(config.AnnouncementChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID must be a Discord snowflake")
	}
	return config, nil
}
