package inactivity

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var discordSnowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)
var messageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

const performanceChannelEnvironment = "DISCORD_BOT_PERFORMANCE_CHANNEL_ID"

// Config controls the inactivity dismissal registry and Discord dashboard.
type Config struct {
	// Enabled activates the registry dashboard and interactions.
	Enabled bool `env:"DISCORD_BOT_INACTIVITY_ENABLED" envDefault:"true"`
	// ChannelID selects the Discord channel containing the registry message.
	ChannelID string `env:"DISCORD_BOT_INACTIVITY_CHANNEL_ID"`
	// MessageKey is the stable managed-message key.
	MessageKey string `env:"DISCORD_BOT_INACTIVITY_MESSAGE_KEY" envDefault:"inactivity-dismissals"`
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
	if config.ChannelID == "" {
		config.ChannelID = strings.TrimSpace(os.Getenv(performanceChannelEnvironment))
	}
	config.MessageKey = strings.TrimSpace(config.MessageKey)
	config.AnnouncementChannelID = strings.TrimSpace(config.AnnouncementChannelID)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_INACTIVITY_REFRESH_INTERVAL must be positive")
	}
	if !config.Enabled {
		return config, nil
	}
	if !discordSnowflakePattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_INACTIVITY_CHANNEL_ID must be a Discord snowflake")
	}
	if !messageKeyPattern.MatchString(config.MessageKey) {
		return Config{}, fmt.Errorf("DISCORD_BOT_INACTIVITY_MESSAGE_KEY is invalid")
	}
	if config.AnnouncementChannelID != "" && !discordSnowflakePattern.MatchString(config.AnnouncementChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID must be a Discord snowflake")
	}
	return config, nil
}
