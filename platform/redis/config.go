// Package redis contains reusable Redis infrastructure.
package redis

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contains Redis connection settings.
type Config struct {
	// Address is the Redis server address.
	Address string `env:"ADDR" envDefault:"127.0.0.1:6379"`
	// Username is the Redis ACL username.
	Username string `env:"USERNAME" envDefault:""`
	// Password is the Redis password.
	Password string `env:"PASSWORD" envDefault:""`
	// Database is the selected Redis database.
	Database int `env:"DB" envDefault:"0"`
	// DialTimeout limits connection establishment.
	DialTimeout time.Duration `env:"DIAL_TIMEOUT" envDefault:"5s"`
	// HealthTimeout limits health probes.
	HealthTimeout time.Duration `env:"HEALTH_TIMEOUT" envDefault:"2s"`
}

// LoadConfig reads Redis configuration from DISCORD_BOT_REDIS_* variables.
func LoadConfig() (Config, error) {
	return env.ParseAsWithOptions[Config](env.Options{Prefix: "DISCORD_BOT_REDIS_"})
}
