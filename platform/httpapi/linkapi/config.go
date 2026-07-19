// Package linkapi exposes Discord account-link HTTP routes.
package linkapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/pixelados-net/discord-bot/platform/discordoauth"
)

// Config contains the allowlisted OAuth completion destinations.
type Config struct {
	// CompletionURLs maps stable caller keys to absolute browser return URLs.
	CompletionURLs map[string]string
	publicURL      string
}

type environmentConfig struct {
	CompletionURLs string `env:"DISCORD_BOT_OAUTH_COMPLETION_URLS"`
}

// LoadConfig reads and validates OAuth browser route configuration.
func LoadConfig(oauthConfig discordoauth.Config) (Config, error) {
	environment, err := env.ParseAs[environmentConfig]()
	if err != nil {
		return Config{}, err
	}
	config := Config{CompletionURLs: map[string]string{}, publicURL: oauthConfig.PublicURL}
	if strings.TrimSpace(environment.CompletionURLs) != "" {
		if err = json.Unmarshal([]byte(environment.CompletionURLs), &config.CompletionURLs); err != nil {
			return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_COMPLETION_URLS must be a JSON object: %w", err)
		}
	}
	if !oauthConfig.Enabled {
		return config, nil
	}
	if len(config.CompletionURLs) == 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_OAUTH_COMPLETION_URLS is required when OAuth is enabled")
	}
	for key, destination := range config.CompletionURLs {
		if !completionKeyPattern.MatchString(key) {
			return Config{}, fmt.Errorf("invalid OAuth completion key %q", key)
		}
		parsed, parseErr := url.Parse(destination)
		if parseErr != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" ||
			!secureOrLoopback(parsed) {
			return Config{}, fmt.Errorf("OAuth completion URL %q must be HTTPS or localhost HTTP", key)
		}
		config.CompletionURLs[key] = parsed.String()
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

func (config Config) startURL(intentID string) string {
	return strings.TrimRight(config.publicURL, "/") + "/oauth/discord/start/" + url.PathEscape(intentID)
}

func (config Config) completionURL(key string, code string) (string, error) {
	destination, ok := config.CompletionURLs[key]
	if !ok {
		return "", fmt.Errorf("unknown completion key")
	}
	parsed, err := url.Parse(destination)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("code", code)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
