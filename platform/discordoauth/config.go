// Package discordoauth provides the Discord OAuth HTTP adapter.
package discordoauth

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var clientIDPattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// Config contains Discord OAuth client and public callback settings.
type Config struct {
	// Enabled controls whether account-link OAuth may be used.
	Enabled bool `env:"DISCORD_BOT_OAUTH_ENABLED" envDefault:"false"`
	// ClientID identifies the Discord application.
	ClientID string `env:"DISCORD_BOT_OAUTH_CLIENT_ID"`
	// ClientSecret authenticates confidential OAuth token operations.
	ClientSecret string `env:"DISCORD_BOT_OAUTH_CLIENT_SECRET"`
	// PublicURL is the externally reachable base URL for browser routes.
	PublicURL string `env:"DISCORD_BOT_OAUTH_PUBLIC_URL"`
	// HTTPTimeout bounds every Discord OAuth HTTP operation.
	HTTPTimeout time.Duration `env:"DISCORD_BOT_OAUTH_HTTP_TIMEOUT" envDefault:"10s"`
}

// CallbackURL returns the exact Discord developer portal redirect URL.
func (config Config) CallbackURL() string {
	return strings.TrimRight(config.PublicURL, "/") + "/oauth/discord/callback"
}

// LoadConfig reads and validates optional Discord OAuth configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	config.PublicURL = strings.TrimRight(strings.TrimSpace(config.PublicURL), "/")
	if !config.Enabled {
		return config, nil
	}
	if !clientIDPattern.MatchString(config.ClientID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_CLIENT_ID must be a Discord snowflake")
	}
	if config.ClientSecret == "" {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_CLIENT_SECRET is required when OAuth is enabled")
	}
	publicURL, parseErr := url.Parse(config.PublicURL)
	if parseErr != nil || publicURL.Host == "" || publicURL.User != nil || publicURL.RawQuery != "" ||
		publicURL.Fragment != "" || !secureOrLoopback(publicURL) {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_PUBLIC_URL must be HTTPS or localhost HTTP without query or fragment")
	}
	if config.HTTPTimeout <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_HTTP_TIMEOUT must be positive")
	}
	return config, nil
}

func secureOrLoopback(candidate *url.URL) bool {
	if candidate.Scheme == "https" {
		return true
	}
	address := net.ParseIP(candidate.Hostname())
	return candidate.Scheme == "http" && (candidate.Hostname() == "localhost" || address != nil && address.IsLoopback())
}
