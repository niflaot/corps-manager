package performance

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var discordSnowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)
var messageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Config controls business earnings collection and its Discord dashboard.
type Config struct {
	// Enabled activates collection and dashboard updates.
	Enabled bool `env:"DISCORD_BOT_PERFORMANCE_ENABLED" envDefault:"false"`
	// Endpoint is the sarp-scrapper POST /api/query URL.
	Endpoint string `env:"DISCORD_BOT_PERFORMANCE_ENDPOINT"`
	// EndpointToken optionally authenticates access to the sarp-scrapper tunnel.
	EndpointToken string `env:"DISCORD_BOT_PERFORMANCE_ENDPOINT_TOKEN"`
	// BusinessID selects the SARP business to monitor.
	BusinessID int64 `env:"DISCORD_BOT_PERFORMANCE_BUSINESS_ID"`
	// ChannelID selects the Discord dashboard channel.
	ChannelID string `env:"DISCORD_BOT_PERFORMANCE_CHANNEL_ID"`
	// MessageKey is the stable managed-message key.
	MessageKey string `env:"DISCORD_BOT_PERFORMANCE_MESSAGE_KEY" envDefault:"business-performance"`
	// Interval controls data refresh frequency.
	Interval time.Duration `env:"DISCORD_BOT_PERFORMANCE_INTERVAL" envDefault:"6h"`
	// CutoffWeekday is the weekly period boundary.
	CutoffWeekday time.Weekday `env:"-"`
	// Timezone is used for weekly period boundaries.
	Timezone *time.Location `env:"-"`
	// HTTPTimeout bounds upstream calls.
	HTTPTimeout time.Duration `env:"DISCORD_BOT_PERFORMANCE_HTTP_TIMEOUT" envDefault:"30s"`
	// MaxResponseBytes bounds upstream response bodies.
	MaxResponseBytes int64 `env:"DISCORD_BOT_PERFORMANCE_MAX_RESPONSE_BYTES" envDefault:"2097152"`
	// HistoryLimit bounds retained weekly cuts.
	HistoryLimit int `env:"DISCORD_BOT_PERFORMANCE_HISTORY_LIMIT" envDefault:"104"`
}

type rawConfig struct {
	Config
	CutoffWeekday string `env:"DISCORD_BOT_PERFORMANCE_CUTOFF_WEEKDAY" envDefault:"Tuesday"`
	Timezone      string `env:"DISCORD_BOT_PERFORMANCE_TIMEZONE" envDefault:"America/Bogota"`
}

// LoadConfig reads and validates performance configuration.
func LoadConfig() (Config, error) {
	raw, err := env.ParseAs[rawConfig]()
	if err != nil {
		return Config{}, err
	}
	config := raw.Config
	config.Endpoint = strings.TrimSpace(config.Endpoint)
	config.EndpointToken = strings.TrimSpace(config.EndpointToken)
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	config.MessageKey = strings.TrimSpace(config.MessageKey)
	config.CutoffWeekday, err = parseWeekday(raw.CutoffWeekday)
	if err != nil {
		return Config{}, err
	}
	config.Timezone, err = time.LoadLocation(strings.TrimSpace(raw.Timezone))
	if err != nil {
		return Config{}, fmt.Errorf("load performance timezone: %w", err)
	}
	if !config.Enabled {
		return config, nil
	}
	endpoint, parseErr := url.ParseRequestURI(config.Endpoint)
	if parseErr != nil || endpoint.Host == "" || endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return Config{}, fmt.Errorf("DISCORD_BOT_PERFORMANCE_ENDPOINT must be an absolute HTTP(S) URL")
	}
	if config.BusinessID <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_PERFORMANCE_BUSINESS_ID must be positive")
	}
	if !discordSnowflakePattern.MatchString(config.ChannelID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_PERFORMANCE_CHANNEL_ID must be a Discord snowflake")
	}
	if config.Interval <= 0 || config.HTTPTimeout <= 0 || config.MaxResponseBytes <= 0 || config.HistoryLimit <= 0 {
		return Config{}, fmt.Errorf("performance durations and limits must be positive")
	}
	if !messageKeyPattern.MatchString(config.MessageKey) {
		return Config{}, fmt.Errorf("DISCORD_BOT_PERFORMANCE_MESSAGE_KEY is invalid")
	}
	return config, nil
}

func parseWeekday(value string) (time.Weekday, error) {
	for day := time.Sunday; day <= time.Saturday; day++ {
		if strings.EqualFold(day.String(), strings.TrimSpace(value)) {
			return day, nil
		}
	}
	return 0, fmt.Errorf("DISCORD_BOT_PERFORMANCE_CUTOFF_WEEKDAY is invalid")
}
