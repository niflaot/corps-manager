package messages

import (
	"strings"
	"testing"
)

func TestPayloadHashIsCanonicalAndObservable(t *testing.T) {
	payload := validDefinition().Payload
	first, err := payload.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	payload.AllowedMentions.Users = []string{"123"}
	second, _ := payload.Hash()
	if first != second {
		t.Fatalf("allowed mentions changed observable hash")
	}
	payload.Components[0] = Component(`{"type":10,"content":"changed"}`)
	third, _ := payload.Hash()
	if third == first {
		t.Fatal("component change did not change hash")
	}
}

func TestPayloadHashIgnoresDiscordAssignedComponentIDs(t *testing.T) {
	withoutID := Payload{Components: []Component{Component(`{"type":1,"components":[{"type":2,"style":1,"label":"Join","custom_id":"join"}]}`)}}
	withID := Payload{Components: []Component{Component(`{"type":1,"id":1,"components":[{"type":2,"id":2,"style":1,"label":"Join","custom_id":"join"}]}`)}}
	first, firstErr := withoutID.Hash()
	second, secondErr := withID.Hash()
	if firstErr != nil || secondErr != nil || first != second {
		t.Fatalf("hashes differ: %q %q, errors = %v %v", first, second, firstErr, secondErr)
	}
}

func TestDefinitionValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Definition)
		want   string
	}{
		{"key", func(value *Definition) { value.Key = "Bad Key" }, "key"},
		{"snowflake", func(value *Definition) { value.ChannelID = "channel" }, "snowflake"},
		{"empty", func(value *Definition) { value.Payload.Components = nil }, "at least"},
		{"text", func(value *Definition) {
			value.Payload.Components[0] = Component(`{"type":10,"content":"` + strings.Repeat("x", 4001) + `"}`)
		}, "4000"},
		{"row", func(value *Definition) {
			value.Payload.Components = []Component{Component(`{"type":1,"components":[]}`)}
		}, "1 to 5"},
		{"mentions", func(value *Definition) {
			value.Payload.AllowedMentions.Parse = []string{"roles"}
			value.Payload.AllowedMentions.Roles = []string{"123"}
		}, "overlap"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := validDefinition()
			test.mutate(&definition)
			if err := definition.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func validDefinition() Definition {
	return Definition{Key: "rules", GuildID: "123456789", ChannelID: "987654321", Payload: Payload{
		Components: []Component{Component(`{"type":10,"content":"Rules"}`)}, AllowedMentions: AllowedMentions{Parse: []string{}},
	}}
}
