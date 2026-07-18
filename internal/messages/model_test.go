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

func TestPayloadHashIgnoresDiscordDefaultFalseFields(t *testing.T) {
	minimal := Payload{Components: []Component{Component(`{"type":17,"components":[{"type":10,"content":"Welcome"}]}`)}}
	discordResponse := Payload{Components: []Component{Component(`{"type":17,"id":1,"spoiler":false,"components":[{"type":10,"id":2,"content":"Welcome"},{"type":1,"id":3,"components":[{"type":2,"id":4,"style":3,"label":"Join","custom_id":"join","disabled":false}]}]}`)}}
	desired := Payload{Components: []Component{Component(`{"type":17,"components":[{"type":10,"content":"Welcome"},{"type":1,"components":[{"type":2,"style":3,"label":"Join","custom_id":"join"}]}]}`)}}
	minimalHash, minimalError := minimal.Hash()
	responseHash, responseError := discordResponse.Hash()
	desiredHash, desiredError := desired.Hash()
	if minimalError != nil || responseError != nil || desiredError != nil || responseHash != desiredHash || minimalHash == desiredHash {
		t.Fatalf("hashes = %q %q %q, errors = %v %v %v", minimalHash, responseHash, desiredHash,
			minimalError, responseError, desiredError)
	}
}

func TestPayloadHashIgnoresMediaGalleryDefaultSpoiler(t *testing.T) {
	minimal := Payload{Components: []Component{Component(`{"type":12,"items":[{"media":{"url":"https://example.com/image.png"}}]}`)}}
	discordResponse := Payload{Components: []Component{Component(`{"type":12,"id":1,"items":[{"media":{"url":"https://example.com/image.png"},"spoiler":false}]}`)}}
	minimalHash, minimalError := minimal.Hash()
	responseHash, responseError := discordResponse.Hash()
	if minimalError != nil || responseError != nil || minimalHash != responseHash {
		t.Fatalf("hashes = %q %q, errors = %v %v", minimalHash, responseHash, minimalError, responseError)
	}
}

func TestPayloadHashIgnoresDiscordThumbnailMetadata(t *testing.T) {
	desired := Payload{Components: []Component{Component(`{"type":11,"media":{"url":"https://example.com/image.png"},"description":"Pixelados"}`)}}
	observed := Payload{Components: []Component{Component(`{"type":11,"id":4,"media":{"id":"asset","url":"https://example.com/image.png","proxy_url":"https://cdn.example.com/image.png","width":1920,"height":1920},"description":"Pixelados","spoiler":false}`)}}
	desiredHash, desiredError := desired.Hash()
	observedHash, observedError := observed.Hash()
	if desiredError != nil || observedError != nil || desiredHash != observedHash {
		t.Fatalf("hashes = %q %q, errors = %v %v", desiredHash, observedHash, desiredError, observedError)
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
