package security

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/corps-manager/internal/messages"
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

func TestRuntimeSettingsOverrideEnvironmentAndDriveRefresh(t *testing.T) {
	store := &messageStub{records: map[string]messages.Record{"verification": verificationRecord()}}
	gateway := &gatewayStub{trapID: "654"}
	runtime := &settingsStub{values: map[string]string{}}
	service := NewService(Config{MessageKey: "verification"}, "123", store, gateway, runtime)
	if err := service.Refresh(context.Background()); err != nil || store.mutations != 0 {
		t.Fatalf("unconfigured refresh mutated: %v", err)
	}
	enable, channel, role := true, "456", "321"
	config, err := service.Configure(context.Background(), Update{VerificationEnabled: &enable, ChannelID: &channel, RoleID: &role, AntibotEnabled: &enable})
	if err != nil || !config.VerificationReady() || !config.AntibotEnabled {
		t.Fatalf("config=%+v err=%v", config, err)
	}
	if err := service.Refresh(context.Background()); err != nil || store.mutations == 0 || !service.IsTrap("123", "654") {
		t.Fatalf("runtime settings were not applied: %v", err)
	}
	disable := false
	if _, err := service.Configure(context.Background(), Update{AntibotEnabled: &disable}); err != nil {
		t.Fatal(err)
	}
	if err := service.Refresh(context.Background()); err != nil || service.IsTrap("123", "654") {
		t.Fatalf("disabled trap stayed armed: %v", err)
	}
}

func TestConfigureRejectsIncompleteOrMalformedChanges(t *testing.T) {
	service := NewService(Config{MessageKey: "verification"}, "123", &messageStub{}, &gatewayStub{}, &settingsStub{values: map[string]string{}})
	enable, bad, reserved := true, "abc", TrapMessageKey
	for _, update := range []Update{{}, {VerificationEnabled: &enable}, {ChannelID: &bad}, {MessageKey: &reserved}} {
		if _, err := service.Configure(context.Background(), update); !errors.Is(err, ErrInvalidSettings) {
			t.Fatalf("update %+v accepted: %v", update, err)
		}
	}
	if _, err := NewService(Config{}, "123", &messageStub{}, &gatewayStub{}).Configure(context.Background(), Update{VerificationEnabled: &enable}); !errors.Is(err, ErrSettingsUnavailable) {
		t.Fatal(err)
	}
}
