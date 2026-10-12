// Package messagesdiscord manages static messages through slash commands.
package messagesdiscord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/messages"
)

// maxPayloadBytes bounds JSON accepted from attachments and modals.
const maxPayloadBytes = 256 << 10

// buttonComponentType is the Components V2 button type.
const buttonComponentType = 2

// parsePayload decodes, normalizes, and validates a message body.
// It accepts either the API payload object or a bare array of components.
func parsePayload(raw []byte) (messages.Payload, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return messages.Payload{}, errors.New("el JSON está vacío")
	}
	if len(raw) > maxPayloadBytes {
		return messages.Payload{}, fmt.Errorf("el JSON supera %d KiB", maxPayloadBytes>>10)
	}
	var payload messages.Payload
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &payload.Components); err != nil {
			return messages.Payload{}, fmt.Errorf("JSON inválido: %w", err)
		}
	} else {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			return messages.Payload{}, fmt.Errorf("JSON inválido: %w", err)
		}
		if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
			return messages.Payload{}, errors.New("JSON inválido: hay contenido después del objeto")
		}
	}
	payload = payload.Normalize()
	if err := payload.Validate(); err != nil {
		return messages.Payload{}, err
	}
	return payload, nil
}

// previewComponents converts a payload into SDK components with every button disabled.
func previewComponents(payload messages.Payload) ([]discordgo.MessageComponent, error) {
	components := make([]discordgo.MessageComponent, len(payload.Components))
	for index, raw := range payload.Components {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		disableButtons(value)
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if components[index], err = discordgo.MessageComponentFromJSON(encoded); err != nil {
			return nil, err
		}
	}
	return components, nil
}

func disableButtons(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	if kind, _ := object["type"].(float64); kind == buttonComponentType {
		object["disabled"] = true
	}
	if children, ok := object["components"].([]any); ok {
		for _, child := range children {
			disableButtons(child)
		}
	}
	disableButtons(object["accessory"])
}
