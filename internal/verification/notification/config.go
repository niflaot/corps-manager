package notification

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config controls durable verification notification delivery.
type Config struct {
	// Interval controls the periodic outbox sweep.
	Interval time.Duration `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_INTERVAL" envDefault:"15s"`
	// BatchSize bounds one claimed delivery batch.
	BatchSize int `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_BATCH_SIZE" envDefault:"25"`
	// Workers bounds concurrent Discord deliveries.
	Workers int `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_WORKERS" envDefault:"4"`
	// LeaseDuration controls recovery after a worker stops.
	LeaseDuration time.Duration `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_LEASE_DURATION" envDefault:"2m"`
	// MaxAttempts moves repeatedly failing deliveries to the dead-letter state.
	MaxAttempts int `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_MAX_ATTEMPTS" envDefault:"8"`
	// RetryBase is the first retry delay.
	RetryBase time.Duration `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_RETRY_BASE" envDefault:"30s"`
	// RetryMax caps exponential retry delays.
	RetryMax time.Duration `env:"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_RETRY_MAX" envDefault:"30m"`
}

// LoadConfig reads verification notification configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	if config.Interval <= 0 || config.BatchSize <= 0 || config.Workers <= 0 || config.LeaseDuration <= 0 ||
		config.MaxAttempts <= 0 || config.RetryBase <= 0 || config.RetryMax < config.RetryBase {
		return Config{}, fmt.Errorf("invalid verification notification configuration")
	}
	return config, nil
}
