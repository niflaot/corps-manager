package security

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const mutationPrefix = "security-control"

func (service *Service) refreshVerification(ctx context.Context) error {
	record, err := service.messages.Get(ctx, service.config.MessageKey)
	if err != nil {
		return fmt.Errorf("read verification message: %w", err)
	}
	if record.State == messages.StateArchived || record.GuildID != service.guildID {
		return ErrNotReady
	}
	definition := record.Definition
	definition.ChannelID = service.config.ChannelID
	definition.Payload, err = withVerificationButton(definition.Payload)
	if err != nil {
		return err
	}
	return service.publish(ctx, record, definition)
}

func (service *Service) publish(ctx context.Context, record messages.Record, definition messages.Definition) error {
	definition.Payload = definition.Payload.Normalize()
	hash, err := definition.Payload.Hash()
	if err != nil {
		return err
	}
	if record.Key != "" && record.State != messages.StateArchived && record.ChannelID == definition.ChannelID && record.DesiredHash == hash {
		return nil
	}
	key := fmt.Sprintf("%s:%s:%d:%s:%s", mutationPrefix, definition.Key, record.Revision, definition.ChannelID, hash[:16])
	if record.Key == "" {
		_, err = service.messages.Create(ctx, definition, key)
	} else {
		_, err = service.messages.Replace(ctx, definition.Key, record.Revision, definition, key)
	}
	return err
}

func withVerificationButton(payload messages.Payload) (messages.Payload, error) {
	var components []map[string]any
	for _, raw := range payload.Components {
		var component map[string]any
		if err := json.Unmarshal(raw, &component); err != nil {
			return messages.Payload{}, err
		}
		if cleanComponent(component) {
			components = append(components, component)
		}
	}
	payload.Components = nil
	for _, component := range components {
		raw, err := json.Marshal(component)
		if err != nil {
			return messages.Payload{}, err
		}
		payload.Components = append(payload.Components, raw)
	}
	button, err := json.Marshal(map[string]any{"type": 1, "components": []any{map[string]any{
		"type": 2, "style": 3, "label": "Verificarme", "custom_id": VerifyButtonID,
	}}})
	if err != nil {
		return messages.Payload{}, err
	}
	payload.Components = append(payload.Components, button)
	return payload, nil
}

func cleanComponent(component map[string]any) bool {
	if component["custom_id"] == VerifyButtonID {
		return false
	}
	if children, ok := component["components"].([]any); ok {
		kept := make([]any, 0, len(children))
		for _, child := range children {
			object, ok := child.(map[string]any)
			if !ok || cleanComponent(object) {
				kept = append(kept, child)
			}
		}
		component["components"] = kept
		if len(kept) == 0 {
			return false
		}
	}
	if accessory, ok := component["accessory"].(map[string]any); ok && accessory["custom_id"] == VerifyButtonID {
		// Sections require accessories; retain their text without nesting containers.
		var paragraphs []string
		children, _ := component["components"].([]any)
		for _, child := range children {
			object, _ := child.(map[string]any)
			if content, ok := object["content"].(string); ok {
				paragraphs = append(paragraphs, content)
			}
		}
		clear(component)
		component["type"] = float64(10)
		component["content"] = strings.Join(paragraphs, "\n")
	}
	return true
}
