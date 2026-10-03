package security

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/niflaot/corps-manager/internal/messages"
)

func TestVerificationControlRepairsDuplicateOrDisabledButtons(t *testing.T) {
	payload := messages.Payload{Components: []messages.Component{[]byte(`{"type":17,"components":[{"type":10,"content":"Conserva estas reglas"},{"type":1,"components":[{"type":2,"style":1,"label":"Old","custom_id":"security:verify","disabled":true},{"type":2,"style":1,"label":"Other","custom_id":"other"}]}]}`)}}
	repaired, err := withVerificationButton(payload)
	if err != nil {
		t.Fatal(err)
	}
	definition := messages.Definition{Key: "rules", GuildID: "123", ChannelID: "456", Payload: repaired}
	if err := definition.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(repaired)
	if strings.Count(string(encoded), VerifyButtonID) != 1 || !strings.Contains(string(encoded), "Conserva estas reglas") || !strings.Contains(string(encoded), "other") {
		t.Fatalf("repaired: %s", encoded)
	}
	again, err := withVerificationButton(repaired)
	if err != nil {
		t.Fatal(err)
	}
	firstHash, _ := repaired.Hash()
	secondHash, _ := again.Hash()
	if firstHash != secondHash {
		t.Fatal("control insertion is not stable")
	}
}

func TestSecurityConfigRequiresBindingsOnlyWhenEnabled(t *testing.T) {
	t.Setenv("DISCORD_BOT_VERIFICATION_ENABLED", "false")
	t.Setenv("DISCORD_BOT_SECURITY_REFRESH_INTERVAL", "1m")
	if _, err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DISCORD_BOT_VERIFICATION_ENABLED", "true")
	t.Setenv("DISCORD_BOT_VERIFICATION_CHANNEL_ID", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("accepted missing channel")
	}
	t.Setenv("DISCORD_BOT_VERIFICATION_CHANNEL_ID", "123")
	t.Setenv("DISCORD_BOT_VERIFICATION_ROLE_ID", "456")
	t.Setenv("DISCORD_BOT_VERIFICATION_MESSAGE_KEY", TrapMessageKey)
	if _, err := LoadConfig(); err == nil {
		t.Fatal("accepted reserved key")
	}
	t.Setenv("DISCORD_BOT_VERIFICATION_MESSAGE_KEY", "verification")
	if _, err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DISCORD_BOT_SECURITY_REFRESH_INTERVAL", "0s")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("accepted zero interval")
	}
}

func TestVerificationRepairsButtonInsideSectionWithoutInvalidNesting(t *testing.T) {
	payload := messages.Payload{Components: []messages.Component{[]byte(`{"type":17,"components":[{"type":9,"components":[{"type":10,"content":"Mis reglas"}],"accessory":{"type":2,"style":1,"label":"Old","custom_id":"security:verify"}}]}`)}}
	repaired, err := withVerificationButton(payload)
	if err != nil {
		t.Fatal(err)
	}
	definition := messages.Definition{Key: "verification", GuildID: "123", ChannelID: "456", Payload: repaired}
	if err := definition.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(repaired)
	if !strings.Contains(string(encoded), "Mis reglas") || strings.Count(string(encoded), VerifyButtonID) != 1 {
		t.Fatalf("payload=%s", encoded)
	}
}
