package agreements

import (
	"context"
	"errors"
	"testing"
)

type settingsStub struct{ values map[string]string }

func (stub *settingsStub) List(context.Context, string) (map[string]string, error) {
	return stub.values, nil
}
func (stub *settingsStub) SetMany(_ context.Context, values map[string]string) error {
	for key, value := range values {
		stub.values[key] = value
	}
	return nil
}

func TestRuntimeSettingsEnableAgreementsAndOverrideEnvironment(t *testing.T) {
	service := NewService(Config{Enabled: true}, nil, nil, "123", &settingsStub{values: map[string]string{}})
	if service.Current().Enabled {
		t.Fatal("agreements enabled without a control channel")
	}
	enabled, channel := true, "456"
	config, err := service.Configure(context.Background(), Update{Enabled: &enabled, ControlChannelID: &channel})
	if err != nil || !config.Enabled || service.Current().ControlChannelID != "456" {
		t.Fatalf("config=%+v err=%v", config, err)
	}
	off := false
	if _, err := service.Configure(context.Background(), Update{Enabled: &off}); err != nil || service.Current().Enabled {
		t.Fatalf("disable failed: %v", err)
	}
}

func TestConfigureRejectsInvalidChanges(t *testing.T) {
	service := NewService(Config{}, nil, nil, "123", &settingsStub{values: map[string]string{}})
	enabled, bad := true, "abc"
	for _, update := range []Update{{}, {Enabled: &enabled}, {ControlChannelID: &bad}} {
		if _, err := service.Configure(context.Background(), update); !errors.Is(err, ErrInvalidSettings) {
			t.Fatalf("update %+v accepted: %v", update, err)
		}
	}
	if _, err := NewService(Config{}, nil, nil, "123").Configure(context.Background(), Update{Enabled: &enabled}); !errors.Is(err, ErrSettingsUnavailable) {
		t.Fatal(err)
	}
}
