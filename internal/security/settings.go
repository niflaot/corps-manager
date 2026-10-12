package security

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	// SettingPrefix namespaces every runtime security setting.
	SettingPrefix = "security."
	// SettingVerificationEnabled overrides whether verification is active.
	SettingVerificationEnabled = SettingPrefix + "verification.enabled"
	// SettingVerificationChannel overrides the verification channel snowflake.
	SettingVerificationChannel = SettingPrefix + "verification.channel_id"
	// SettingVerificationRole overrides the role granted after verification.
	SettingVerificationRole = SettingPrefix + "verification.role_id"
	// SettingVerificationMessageKey overrides the managed verification message key.
	SettingVerificationMessageKey = SettingPrefix + "verification.message_key"
	// SettingAntibotEnabled overrides whether the anti-bot trap is active.
	SettingAntibotEnabled = SettingPrefix + "antibot.enabled"
)

var (
	// ErrSettingsUnavailable reports that no runtime settings store is configured.
	ErrSettingsUnavailable = errors.New("runtime settings are unavailable")
	// ErrInvalidSettings reports a rejected runtime configuration change.
	ErrInvalidSettings = errors.New("invalid security settings")
)

// Update contains optional runtime configuration changes; nil fields stay untouched.
type Update struct {
	// VerificationEnabled toggles the verification button.
	VerificationEnabled *bool
	// ChannelID binds the verification channel.
	ChannelID *string
	// RoleID binds the role granted after verification.
	RoleID *string
	// MessageKey binds the managed verification message.
	MessageKey *string
	// AntibotEnabled toggles the anti-bot trap.
	AntibotEnabled *bool
}

// Status describes the effective configuration and the managed verification message.
type Status struct {
	// Config is the effective configuration after runtime overrides.
	Config Config
	// Message is the managed verification message when it exists.
	Message messages.Record
	// MessageFound reports whether the managed verification message exists.
	MessageFound bool
	// TrapChannelID is the currently armed trap channel.
	TrapChannelID string
}

// Effective returns the environment defaults overlaid with runtime settings.
func (service *Service) Effective(ctx context.Context) (Config, error) {
	config := service.config
	if service.settings == nil {
		return config, nil
	}
	values, err := service.settings.List(ctx, SettingPrefix)
	if err != nil {
		return Config{}, fmt.Errorf("read security settings: %w", err)
	}
	for key, value := range values {
		switch key {
		case SettingVerificationEnabled, SettingAntibotEnabled:
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return Config{}, fmt.Errorf("%w: %s", ErrInvalidSettings, key)
			}
			if key == SettingAntibotEnabled {
				config.AntibotEnabled = enabled
			} else {
				config.VerificationEnabled = enabled
			}
		case SettingVerificationChannel:
			config.ChannelID = value
		case SettingVerificationRole:
			config.RoleID = value
		case SettingVerificationMessageKey:
			config.MessageKey = value
		}
	}
	return config, nil
}

// Configure validates and persists runtime overrides, returning the new effective configuration.
func (service *Service) Configure(ctx context.Context, update Update) (Config, error) {
	if service.settings == nil {
		return Config{}, ErrSettingsUnavailable
	}
	config, err := service.Effective(ctx)
	if err != nil {
		return Config{}, err
	}
	values := map[string]string{}
	if update.VerificationEnabled != nil {
		config.VerificationEnabled = *update.VerificationEnabled
		values[SettingVerificationEnabled] = strconv.FormatBool(config.VerificationEnabled)
	}
	if update.AntibotEnabled != nil {
		config.AntibotEnabled = *update.AntibotEnabled
		values[SettingAntibotEnabled] = strconv.FormatBool(config.AntibotEnabled)
	}
	if update.ChannelID != nil {
		config.ChannelID = strings.TrimSpace(*update.ChannelID)
		values[SettingVerificationChannel] = config.ChannelID
	}
	if update.RoleID != nil {
		config.RoleID = strings.TrimSpace(*update.RoleID)
		values[SettingVerificationRole] = config.RoleID
	}
	if update.MessageKey != nil {
		config.MessageKey = strings.TrimSpace(*update.MessageKey)
		values[SettingVerificationMessageKey] = config.MessageKey
	}
	if len(values) == 0 {
		return Config{}, fmt.Errorf("%w: no changes requested", ErrInvalidSettings)
	}
	if err := config.validateBindings(); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	if config.VerificationEnabled && (config.ChannelID == "" || config.RoleID == "") {
		return Config{}, fmt.Errorf("%w: verification needs a channel and a role before it can be enabled", ErrInvalidSettings)
	}
	if err := service.settings.SetMany(ctx, values); err != nil {
		return Config{}, fmt.Errorf("save security settings: %w", err)
	}
	return config, nil
}

// Status reports the effective configuration and the managed verification message.
func (service *Service) Status(ctx context.Context) (Status, error) {
	config, err := service.Effective(ctx)
	if err != nil {
		return Status{}, err
	}
	status := Status{Config: config}
	status.TrapChannelID, _ = service.trap.Load().(string)
	record, err := service.messages.Get(ctx, config.MessageKey)
	switch {
	case err == nil:
		status.Message, status.MessageFound = record, true
	case !errors.Is(err, messages.ErrNotFound):
		return Status{}, err
	}
	return status, nil
}
