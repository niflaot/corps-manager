package inactivity

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	registryAccent = 0xe67e22
	// ButtonAddCustomID identifies the button that opens the add modal.
	ButtonAddCustomID = "inactivity:add"
	// ButtonRemoveCustomID identifies the button that opens the removal modal.
	ButtonRemoveCustomID = "inactivity:remove"
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
	content := "Sin empleados registrados."
	if len(entries) > 0 {
		lines := make([]string, 0, len(entries)+2)
		lines = append(lines, "```text", "Empleado")
		for _, entry := range entries {
			lines = append(lines, entry.Name)
		}
		lines = append(lines, "```")
		content = strings.Join(lines, "\n")
	}
	container := component{Type: 17, Accent: registryAccent, Components: []component{
		{Type: 10, Content: "# 💤 Expulsados por inactividad"},
		{Type: 10, Content: fmt.Sprintf("**Total:** %d\n%s", len(entries), content)},
		{Type: 14, Divider: true, Spacing: 1},
		{Type: 1, Components: []component{
			{Type: 2, Style: 3, Label: "Añadir empleado", CustomID: ButtonAddCustomID},
			{Type: 2, Style: 4, Label: "Retirar empleado", CustomID: ButtonRemoveCustomID},
		}},
	}}
	encoded, err := json.Marshal(container)
	if err != nil {
		return messages.Definition{}, fmt.Errorf("encode inactivity dashboard: %w", err)
	}
	return messages.Definition{Key: config.MessageKey, GuildID: guildID, ChannelID: config.ChannelID,
		Payload: messages.Payload{Components: []messages.Component{encoded},
			AllowedMentions: messages.AllowedMentions{Parse: []string{}}}}, nil
}
