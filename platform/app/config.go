// Package app contains application-level runtime configuration.
package app

import (
	"fmt"
	"net"
	"strconv"

	"github.com/caarlos0/env/v11"
)

// Environment identifies a runtime environment.
type Environment string

const (
	// EnvironmentDevelopment enables development behavior.
	EnvironmentDevelopment Environment = "development"
	// EnvironmentTest identifies automated test execution.
	EnvironmentTest Environment = "test"
	// EnvironmentProduction enables production behavior.
	EnvironmentProduction Environment = "production"
)

// Config contains application-level runtime settings.
type Config struct {
	// Environment names the runtime environment.
	Environment Environment `env:"ENVIRONMENT" envDefault:"development"`
	// Host is the HTTP bind host.
	Host string `env:"HOST" envDefault:"127.0.0.1"`
	// Port is the HTTP bind port.
	Port int `env:"PORT" envDefault:"3100"`
}

// Address returns the host and port formatted for a listener.
func (config Config) Address() string {
	return net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
}

// IsDevelopment reports whether development behavior is enabled.
func (environment Environment) IsDevelopment() bool {
	return environment == EnvironmentDevelopment
}

// UnmarshalText validates and stores an environment value.
func (environment *Environment) UnmarshalText(value []byte) error {
	parsed := Environment(value)
	if parsed != EnvironmentDevelopment && parsed != EnvironmentTest && parsed != EnvironmentProduction {
		return fmt.Errorf("unsupported environment %q", value)
	}
	*environment = parsed
	return nil
}

// LoadConfig reads application configuration from DISCORD_BOT_* variables.
func LoadConfig() (Config, error) {
	return env.ParseAsWithOptions[Config](env.Options{Prefix: "DISCORD_BOT_"})
}
