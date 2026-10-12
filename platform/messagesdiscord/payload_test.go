package messagesdiscord

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParsePayloadAcceptsObjectAndBareArray(t *testing.T) {
	for _, raw := range []string{
		`{"components":[{"type":10,"content":"Hola"}]}`,
		`[{"type":10,"content":"Hola"}]`,
		` {"components":[{"type":17,"components":[{"type":10,"content":"x"}]}],"allowedMentions":{"parse":[]}} `,
	} {
		if _, err := parsePayload([]byte(raw)); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
	}
}

func TestParsePayloadRejectsInvalidInput(t *testing.T) {
	for name, raw := range map[string]string{
		"empty":        ``,
		"syntax":       `{"components":`,
		"unknown":      `{"componentz":[]}`,
		"trailing":     `{"components":[{"type":10,"content":"a"}]} {}`,
		"no component": `{"components":[]}`,
		"bad shape":    `[{"type":10,"content":"a","extra":1}]`,
		"empty text":   `[{"type":10,"content":""}]`,
		"too large":    `[{"type":10,"content":"` + strings.Repeat("a", maxPayloadBytes) + `"}]`,
	} {
		if _, err := parsePayload([]byte(raw)); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestPreviewDisablesEveryButton(t *testing.T) {
	payload, err := parsePayload([]byte(`[{"type":17,"components":[{"type":9,"components":[{"type":10,"content":"x"}],"accessory":{"type":2,"style":1,"label":"A","custom_id":"a"}},{"type":1,"components":[{"type":2,"style":1,"label":"B","custom_id":"security:verify"},{"type":2,"style":5,"label":"C","url":"https://example.com"}]}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	components, err := previewComponents(payload)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(components)
	if strings.Count(string(encoded), `"disabled":true`) != 3 {
		t.Fatalf("buttons not disabled: %s", encoded)
	}
}
