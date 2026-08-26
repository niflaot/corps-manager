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
	config, err := LoadConfig()
	if err != nil || config.ChannelID != "456" {
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
		Config{ChannelID: "456", MessageKey: "inactivity-dismissals"}, "123")
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
	for _, expected := range []string{"Thomas_Jhonson", ButtonAddCustomID, ButtonRemoveCustomID} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("rendered payload does not contain %q: %s", expected, payload)
		}
	}
}
