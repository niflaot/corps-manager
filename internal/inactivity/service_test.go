package inactivity

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestLoadConfigDefaultsToPerformanceChannel(t *testing.T) {
	t.Setenv("DISCORD_BOT_INACTIVITY_ENABLED", "true")
	t.Setenv("DISCORD_BOT_INACTIVITY_CHANNEL_ID", "")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CHANNEL_ID", "456")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "789")
	config, err := LoadConfig()
	if err != nil || config.ChannelID != "456" || config.AnnouncementChannelID != "789" {
		t.Fatalf("LoadConfig() = %#v, %v", config, err)
	}
}

func TestNormalizeName(t *testing.T) {
	display, normalized, err := normalizeName("  Thomas_Jhonson  ")
	if err != nil || display != "Thomas_Jhonson" || normalized != "thomas_jhonson" {
		t.Fatalf("normalizeName() = %q, %q, %v", display, normalized, err)
	}
	for _, value := range []string{"Thomas", "Thomas Jhonson", "Thomas_Jhonson_II", "Thomas_123"} {
		if _, _, err := normalizeName(value); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("normalizeName(%q) error = %v", value, err)
		}
	}
}

func TestRenderProducesInteractiveManagedMessage(t *testing.T) {
	definition, err := Render([]Entry{{Name: "Thomas_Jhonson"}, {Name: "Andy_Quintero"}},
		Config{ChannelID: "456", MessageKey: "inactivity-dismissals", AnnouncementChannelID: "789"}, "123")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("rendered definition is invalid: %v", err)
	}
	encoded, err := json.Marshal(definition.Payload)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	for _, expected := range []string{ButtonListCustomID, ButtonAddCustomID, ButtonRemoveCustomID, ButtonOpeningCustomID} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("rendered payload does not contain %q: %s", expected, payload)
		}
	}
	if strings.Contains(payload, "Thomas_Jhonson") {
		t.Fatalf("public dashboard exposes inactivity entries: %s", payload)
	}
}

func TestDashboardFingerprintChangesWithOpeningButton(t *testing.T) {
	base := Config{ChannelID: "456", MessageKey: "inactivity-dismissals"}
	withoutButton, err := Render(nil, base, "123")
	if err != nil {
		t.Fatal(err)
	}
	base.AnnouncementChannelID = "789"
	withButton, err := Render(nil, base, "123")
	if err != nil {
		t.Fatal(err)
	}
	withoutFingerprint, err := definitionFingerprint(withoutButton)
	if err != nil {
		t.Fatal(err)
	}
	withFingerprint, err := definitionFingerprint(withButton)
	if err != nil {
		t.Fatal(err)
	}
	if withoutFingerprint == withFingerprint {
		t.Fatalf("opening button did not change fingerprint: %s", withFingerprint)
	}
}
