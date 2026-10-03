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

// LoadConfig validates enabled security features without requiring disabled resources.
func LoadConfig() (Config, error) {
	config := Config{RefreshInterval: DefaultRefreshInterval}
	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}
	config.ChannelID = strings.TrimSpace(config.ChannelID)
	config.RoleID = strings.TrimSpace(config.RoleID)
	config.MessageKey = strings.TrimSpace(config.MessageKey)
	if config.RefreshInterval <= 0 {
		return Config{}, fmt.Errorf("security refresh interval must be positive")
	}
	if config.VerificationEnabled && (!snowflakePattern.MatchString(config.ChannelID) || !snowflakePattern.MatchString(config.RoleID) || !messageKeyPattern.MatchString(config.MessageKey) || config.MessageKey == TrapMessageKey) {
		return Config{}, fmt.Errorf("verification requires a valid channel, role, and nonreserved managed message key")
	}
	return config, nil
}
