// Package postgres contains reusable PostgreSQL infrastructure.
package postgres

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contains PostgreSQL connection settings.
type Config struct {
	// Host is the PostgreSQL server host.
	Host string `env:"HOST" envDefault:"127.0.0.1"`
	// Port is the PostgreSQL server port.
	Port int `env:"PORT" envDefault:"5432"`
	// Database is the PostgreSQL database name.
	Database string `env:"DATABASE" envDefault:"discord_bot"`
	// User is the PostgreSQL username.
	User string `env:"USER" envDefault:"discord_bot"`
	// Password is the PostgreSQL password.
	Password string `env:"PASSWORD" envDefault:"discord_bot"`
	// SSLMode is the PostgreSQL TLS mode.
	SSLMode string `env:"SSL_MODE" envDefault:"disable"`
	// MaxConns is the maximum pool connection count.
	MaxConns int32 `env:"MAX_CONNS" envDefault:"10"`
	// MinConns is the minimum pool connection count.
	MinConns int32 `env:"MIN_CONNS" envDefault:"0"`
	// ConnectTimeout limits connection creation.
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s"`
	// StatementTimeout limits one statement.
	StatementTimeout time.Duration `env:"STATEMENT_TIMEOUT" envDefault:"5s"`
	// HealthTimeout limits health probes.
	HealthTimeout time.Duration `env:"HEALTH_TIMEOUT" envDefault:"2s"`
}

// DSN returns the PostgreSQL connection string.
func (config Config) DSN() string {
	return config.dsn(config.Password)
}

// MaskedDSN returns the PostgreSQL connection string without its secret.
func (config Config) MaskedDSN() string {
	return config.dsn("xxxxx")
}

// LoadConfig reads PostgreSQL configuration from DISCORD_BOT_POSTGRES_* variables.
func LoadConfig() (Config, error) {
	return env.ParseAsWithOptions[Config](env.Options{Prefix: "DISCORD_BOT_POSTGRES_"})
}

func (config Config) dsn(password string) string {
	query := url.Values{}
	query.Set("sslmode", config.SSLMode)
	query.Set("connect_timeout", strconv.Itoa(int(config.ConnectTimeout.Seconds())))
	location := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.User, password),
		Host:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Path:     config.Database,
		RawQuery: query.Encode(),
	}
	return location.String()
}
