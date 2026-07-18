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
	second, err := payload.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if first != second {
		t.Fatalf("allowed mentions changed observable hash: %q != %q", first, second)
	}
	payload.Content = "changed"
	third, _ := payload.Hash()
	if third == first {
		t.Fatal("content change did not change hash")
	}
}

func TestDefinitionValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Definition)
		want   string
	}{
		{name: "key", mutate: func(value *Definition) { value.Key = "Bad Key" }, want: "key"},
		{name: "snowflake", mutate: func(value *Definition) { value.ChannelID = "channel" }, want: "snowflake"},
		{name: "empty", mutate: func(value *Definition) { value.Payload.Content = ""; value.Payload.Embeds = nil }, want: "content"},
		{name: "content", mutate: func(value *Definition) { value.Payload.Content = strings.Repeat("x", 2001) }, want: "2000"},
		{name: "embed total", mutate: func(value *Definition) { value.Payload.Embeds[0].Description = strings.Repeat("x", 4097) }, want: "4096"},
		{name: "url", mutate: func(value *Definition) { value.Payload.Embeds[0].URL = "javascript:bad" }, want: "HTTP"},
		{name: "mentions", mutate: func(value *Definition) {
			value.Payload.AllowedMentions.Parse = []string{"roles"}
			value.Payload.AllowedMentions.Roles = []string{"123"}
		}, want: "overlap"},
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
	return Definition{
		Key: "rules", GuildID: "123456789", ChannelID: "987654321",
		Payload: Payload{
			Content: "Rules", Embeds: []Embed{{Title: "Welcome", Description: "Be kind", Fields: []EmbedField{}}},
			AllowedMentions: AllowedMentions{Parse: []string{}},
		},
	}
}
