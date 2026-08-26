package inactivity

import (
	"encoding/json"
	"fmt"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	registryAccent     = 0xe67e22
	announcementAccent = 0x2ecc71
	registryMessageKey = "inactivity-dismissals"
	openingMessageKey  = "business-opening-control"
	// ButtonListCustomID identifies the button that privately lists entries.
	ButtonListCustomID = "inactivity:list"
	// ButtonAddCustomID identifies the button that opens the add modal.
	ButtonAddCustomID = "inactivity:add"
	// ButtonRemoveCustomID identifies the button that opens the removal modal.
	ButtonRemoveCustomID = "inactivity:remove"
	// ButtonOpeningCustomID identifies the public business-opening action.
	ButtonOpeningCustomID = "inactivity:announce-opening"
)

type component struct {
	Type       int         `json:"type"`
	Content    string      `json:"content,omitempty"`
	Divider    bool        `json:"divider,omitempty"`
	Spacing    int         `json:"spacing,omitempty"`
	Style      int         `json:"style,omitempty"`
	Label      string      `json:"label,omitempty"`
	CustomID   string      `json:"custom_id,omitempty"`
	Components []component `json:"components,omitempty"`
	Accent     int         `json:"accent_color,omitempty"`
}

// Render creates the interactive inactivity registry message definition.
func Render(entries []Entry, config Config, guildID string) (messages.Definition, error) {
	components := []component{
		{Type: 10, Content: "# 💤 Expulsados por inactividad"},
		{Type: 10, Content: fmt.Sprintf("**Total registrado:** %d\nLa lista se consulta de forma privada.", len(entries))},
		{Type: 14, Divider: true, Spacing: 1},
		{Type: 1, Components: []component{
			{Type: 2, Style: 1, Label: "Ver inactivos", CustomID: ButtonListCustomID},
			{Type: 2, Style: 3, Label: "Añadir empleado", CustomID: ButtonAddCustomID},
			{Type: 2, Style: 4, Label: "Retirar empleado", CustomID: ButtonRemoveCustomID},
		}},
	}
	container := component{Type: 17, Accent: registryAccent, Components: components}
	encoded, err := json.Marshal(container)
	if err != nil {
		return messages.Definition{}, fmt.Errorf("encode inactivity dashboard: %w", err)
	}
	return messages.Definition{Key: registryMessageKey, GuildID: guildID, ChannelID: config.ChannelID,
		Payload: messages.Payload{Components: []messages.Component{encoded},
			AllowedMentions: messages.AllowedMentions{Parse: []string{}}}}, nil
}

// RenderOpeningControl creates the separate opening-control message definition.
func RenderOpeningControl(config Config, guildID string) (messages.Definition, error) {
	components := []component{
		{Type: 10, Content: "# 📣 Apertura de Benny's Motor"},
		{Type: 10, Content: "Publica el anuncio de apertura en el canal configurado."},
		{Type: 14, Divider: true, Spacing: 1},
		{Type: 1, Components: []component{
			{Type: 2, Style: 3, Label: "Accionar apertura", CustomID: ButtonOpeningCustomID},
		}},
	}
	container := component{Type: 17, Accent: announcementAccent, Components: components}
	encoded, err := json.Marshal(container)
	if err != nil {
		return messages.Definition{}, fmt.Errorf("encode opening control: %w", err)
	}
	return messages.Definition{Key: openingMessageKey, GuildID: guildID, ChannelID: config.ChannelID,
		Payload: messages.Payload{Components: []messages.Component{encoded},
			AllowedMentions: messages.AllowedMentions{Parse: []string{}}}}, nil
}
