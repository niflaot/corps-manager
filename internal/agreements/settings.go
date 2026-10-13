package agreements

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	// SettingPrefix namespaces every runtime agreements setting.
	SettingPrefix = "agreements."
	// SettingEnabled overrides whether agreements are active.
	SettingEnabled = SettingPrefix + "enabled"
	// SettingControlChannel overrides the control panel channel snowflake.
	SettingControlChannel = SettingPrefix + "control_channel_id"
)

var (
	// ErrSettingsUnavailable reports that no runtime settings store is configured.
	ErrSettingsUnavailable = errors.New("runtime settings are unavailable")
	// ErrInvalidSettings reports a rejected runtime configuration change.
	ErrInvalidSettings = errors.New("invalid agreements settings")
)

// Update contains optional runtime configuration changes; nil fields stay untouched.
type Update struct {
	// Enabled toggles agreement interactions and publishing.
	Enabled *bool
	// ControlChannelID binds the control panel channel.
	ControlChannelID *string
}

// Current returns the effective configuration without touching storage.
func (service *Service) Current() Config { return *service.current.Load() }

// Reload reads runtime settings and refreshes the effective configuration.
func (service *Service) Reload(ctx context.Context) (Config, error) {
	config := service.Current()
	if service.settings == nil {
		return config, nil
	}
	values, err := service.settings.List(ctx, SettingPrefix)
	if err != nil {
		return Config{}, fmt.Errorf("read agreements settings: %w", err)
	}
	base := service.base
	if value, found := values[SettingControlChannel]; found {
		base.ControlChannelID = value
	}
	if value, found := values[SettingEnabled]; found {
		if base.Enabled, err = strconv.ParseBool(value); err != nil {
			return Config{}, fmt.Errorf("%w: %s", ErrInvalidSettings, SettingEnabled)
		}
	}
	base = base.normalized()
	service.current.Store(&base)
	return base, nil
}

// Configure validates and persists runtime overrides, returning the new effective configuration.
func (service *Service) Configure(ctx context.Context, update Update) (Config, error) {
	if service.settings == nil {
		return Config{}, ErrSettingsUnavailable
	}
	if _, err := service.Reload(ctx); err != nil {
		return Config{}, err
	}
	config, values := service.requested(update)
	if len(values) == 0 {
		return Config{}, fmt.Errorf("%w: no changes requested", ErrInvalidSettings)
	}
	if config.ControlChannelID != "" && !snowflakePattern.MatchString(config.ControlChannelID) {
		return Config{}, fmt.Errorf("%w: the control channel must be a Discord snowflake", ErrInvalidSettings)
	}
	if config.Enabled && config.ControlChannelID == "" {
		return Config{}, fmt.Errorf("%w: a control channel is required before enabling agreements", ErrInvalidSettings)
	}
	if err := service.settings.SetMany(ctx, values); err != nil {
		return Config{}, fmt.Errorf("save agreements settings: %w", err)
	}
	return service.Reload(ctx)
}

func (service *Service) requested(update Update) (Config, map[string]string) {
	config, values := service.Current(), map[string]string{}
	if update.Enabled != nil {
		config.Enabled = *update.Enabled
		values[SettingEnabled] = strconv.FormatBool(config.Enabled)
	}
	if update.ControlChannelID != nil {
		config.ControlChannelID = strings.TrimSpace(*update.ControlChannelID)
		values[SettingControlChannel] = config.ControlChannelID
	}
	return config, values
}
