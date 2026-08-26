package inactivity

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestLoadConfigUsesPerformanceChannel(t *testing.T) {
	t.Setenv("DISCORD_BOT_INACTIVITY_ENABLED", "true")
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
		Config{ChannelID: "456", AnnouncementChannelID: "789"}, "123")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("rendered definition is invalid: %v", err)
	}
	if definition.Key != registryMessageKey || definition.ChannelID != "456" {
		t.Fatalf("employee definition = %#v", definition)
	}
	encoded, err := json.Marshal(definition.Payload)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	for _, expected := range []string{ButtonListCustomID, ButtonAddCustomID, ButtonRemoveCustomID} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("rendered payload does not contain %q: %s", expected, payload)
		}
	}
	if strings.Contains(payload, ButtonOpeningCustomID) {
		t.Fatalf("employee dashboard contains opening control: %s", payload)
	}
	if strings.Contains(payload, "Thomas_Jhonson") {
		t.Fatalf("public dashboard exposes inactivity entries: %s", payload)
	}
}

func TestRenderOpeningControlProducesSeparateManagedMessage(t *testing.T) {
	config := Config{ChannelID: "456"}
	definition, err := RenderOpeningControl(config, "123")
	if err != nil {
		t.Fatal(err)
	}
	if definition.Key != openingMessageKey || definition.ChannelID != "456" {
		t.Fatalf("opening definition = %#v", definition)
	}
	encoded, err := json.Marshal(definition.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), ButtonOpeningCustomID) {
		t.Fatalf("opening payload does not contain button: %s", encoded)
	}
}

func TestDashboardFingerprintIncludesAssignment(t *testing.T) {
	definition, err := Render(nil, Config{ChannelID: "456"}, "123")
	if err != nil {
		t.Fatal(err)
	}
	before, err := definitionFingerprint(definition)
	if err != nil {
		t.Fatal(err)
	}
	definition.ChannelID = "789"
	after, err := definitionFingerprint(definition)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatalf("assignment did not change fingerprint: %s", after)
	}
}
