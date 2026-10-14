package classic

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/niflaot/corps-manager/internal/messages"
)

const sample = `{"content":"🔧PRECIOS","embeds":[{"description":"- a > $800","color":15548997,"author":{"name":"PRECIOS:","icon_url":"https://i.imgur.com/a.png"},"thumbnail":{"url":"https://i.imgur.com/a.png"}},{"title":"Cmds","description":"- /x","fields":[{"name":"Uno","value":"1","inline":true}],"image":{"url":"https://i.imgur.com/b.png"},"footer":{"text":"pie"},"timestamp":"2026-01-02T03:04:05Z"}]}`

func TestConvertProducesValidV2Payload(t *testing.T) {
	if !IsClassic([]byte(sample)) || IsClassic([]byte(`{"components":[]}`)) || IsClassic([]byte(`[{"type":10}]`)) {
		t.Fatal("classic detection is wrong")
	}
	payload, err := Convert([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if err := payload.Validate(); err != nil {
		t.Fatalf("converted payload is invalid: %v", err)
	}
	encoded, _ := json.Marshal(payload)
	for _, want := range []string{"🔧PRECIOS", `"accent_color":15548997`, "### PRECIOS:", `"type":11`, "## Cmds", "**Uno**", `"type":12`, "-# pie", "t:1767323045:f"} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("missing %q in %s", want, encoded)
		}
	}
	if len(payload.Components) != 3 {
		t.Fatalf("components = %d", len(payload.Components))
	}
}

func TestConvertKeepsButtonRowsAndMentions(t *testing.T) {
	payload, err := Convert([]byte(`{"content":"hi","username":"x","components":[{"type":1,"components":[{"type":2,"style":5,"label":"Web","url":"https://example.com"}]}],"allowed_mentions":{"parse":["users"]}}`))
	if err != nil || payload.Validate() != nil || len(payload.Components) != 2 || payload.AllowedMentions.Parse[0] != "users" {
		t.Fatalf("payload=%+v err=%v", payload, err)
	}
}

func TestConvertRejectsUnsupportedInput(t *testing.T) {
	long := strings.Repeat("a", maxTextLength+1)
	for name, raw := range map[string]string{
		"empty":      `{"content":"  "}`,
		"unknown":    `{"content":"x","wat":1}`,
		"long":       `{"embeds":[{"description":"` + long + `"}]}`,
		"long hello": `{"content":"` + long + `"}`,
		"syntax":     `{"content":`,
	} {
		if _, err := Convert([]byte(raw)); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if _, err := Convert([]byte(`{"embeds":[{}]}`)); err == nil {
		t.Fatal("empty embed produced a message")
	}
	var _ messages.Payload
}
