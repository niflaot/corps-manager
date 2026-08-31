package customers

import (
	"fmt"
	"net/url"
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
	// PublicURL opens the filterable customer page from Discord.
	PublicURL string `env:"DISCORD_BOT_CUSTOMERS_PUBLIC_URL" envDefault:"https://corps.niflaot.dev/customers"`
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
	config.PublicURL = strings.TrimSpace(config.PublicURL)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_CUSTOMERS_REFRESH_INTERVAL must be positive")
	}
	if config.Enabled && !snowflakePattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_CUSTOMERS_CHANNEL_ID must be a Discord snowflake")
	}
	parsedURL, parseErr := url.ParseRequestURI(config.PublicURL)
	if config.Enabled && (parseErr != nil || parsedURL.Host == "" ||
		parsedURL.Scheme != "https" && parsedURL.Scheme != "http") {
		return Config{}, fmt.Errorf("DISCORD_BOT_CUSTOMERS_PUBLIC_URL must be an absolute HTTP URL")
	}
	return config, nil
}
