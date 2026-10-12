// Package security manages verification and the anti-bot trap.
package security

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

const (
	// VerifyButtonID identifies the verification action.
	VerifyButtonID = "security:verify"
	// TrapMessageKey identifies the durable anti-bot warning.
	TrapMessageKey = "security-antibot"
	// TrapChannelName is the reserved trap channel name.
	TrapChannelName = "no-escribir"
	// TrapWarning is the permanent trap warning.
	TrapWarning = "no escribir, control anti bots"
	// TrapCategoryName identifies the category kept at the bottom of the guild.
	TrapCategoryName = "Control anti bots"
	// DefaultRefreshInterval is the default integrity check interval.
	DefaultRefreshInterval = time.Minute
	// DefaultMessageKey is the default managed verification message key.
	DefaultMessageKey = "verification"
)

var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)
var messageKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Config selects verification resources and security scheduling.
type Config struct {
	// VerificationEnabled activates the verification button; disabled by default.
	VerificationEnabled bool `env:"DISCORD_BOT_VERIFICATION_ENABLED"`
	// ChannelID is the verification channel in the configured guild.
	ChannelID string `env:"DISCORD_BOT_VERIFICATION_CHANNEL_ID"`
	// MessageKey identifies the user-authored managed verification message.
	MessageKey string `env:"DISCORD_BOT_VERIFICATION_MESSAGE_KEY"`
	// RoleID is the role granted after verification.
	RoleID string `env:"DISCORD_BOT_VERIFICATION_ROLE_ID"`
	// AntibotEnabled activates channel reconciliation and automatic bans; disabled by default.
	AntibotEnabled bool `env:"DISCORD_BOT_ANTIBOT_ENABLED"`
	// RefreshInterval defaults to one minute.
	RefreshInterval time.Duration `env:"DISCORD_BOT_SECURITY_REFRESH_INTERVAL"`
}

// validateBindings verifies the format of every configured binding.
func (config Config) validateBindings() error {
	if config.ChannelID != "" && !snowflakePattern.MatchString(config.ChannelID) || config.RoleID != "" && !snowflakePattern.MatchString(config.RoleID) {
		return fmt.Errorf("verification channel and role must be Discord snowflakes")
	}
	if !messageKeyPattern.MatchString(config.MessageKey) || config.MessageKey == TrapMessageKey {
		return fmt.Errorf("verification requires a nonreserved managed message key")
	}
	return nil
}

// VerificationReady reports whether verification is enabled and fully bound.
func (config Config) VerificationReady() bool {
	return config.VerificationEnabled && config.ChannelID != "" && config.RoleID != ""
}

// LoadConfig reads the environment defaults; runtime settings may override them.
func LoadConfig() (Config, error) {
	config := Config{RefreshInterval: DefaultRefreshInterval, MessageKey: DefaultMessageKey}
	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	config.RoleID = strings.TrimSpace(config.RoleID)
	config.MessageKey = strings.TrimSpace(config.MessageKey)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("security refresh interval must be positive")
	}
	if config.MessageKey == "" {
		config.MessageKey = DefaultMessageKey
	}
	if err := config.validateBindings(); err != nil {
		return Config{}, err
	}
	return config, nil
}
