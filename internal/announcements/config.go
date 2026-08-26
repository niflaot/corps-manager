package announcements

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var channelPattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Config controls business-opening announcements and their cooldown.
type Config struct {
	// ChannelID receives public business-opening announcements.
	ChannelID string `env:"DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID"`
	// Cooldown blocks repeated announcements for this duration.
	Cooldown time.Duration `env:"DISCORD_BOT_ANNOUNCEMENT_COOLDOWN" envDefault:"30m"`
}

// LoadConfig reads and validates announcement configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	if config.Cooldown <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_ANNOUNCEMENT_COOLDOWN must be positive")
	}
	if config.ChannelID != "" && !channelPattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID must be a Discord snowflake")
	}
	return config, nil
}
