package agreements

import (
	"encoding/json"
	"fmt"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	agreementsListMessageKey    = "business-agreements"
	agreementsControlMessageKey = "business-agreements-control"
	agreementsAccent            = 0x9b59b6
	// ButtonAddCustomID identifies the add-agreement action.
	ButtonAddCustomID = "agreements:add"
	// ButtonListCustomID identifies the private complete list action.
	ButtonListCustomID = "agreements:list"
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
	Accessory  *component  `json:"accessory,omitempty"`
	Media      *media      `json:"media,omitempty"`
	Accent     int         `json:"accent_color,omitempty"`
}

type media struct {
	URL string `json:"url"`
}

// Render creates the public agreement list and performance-channel control panel.
func Render(items []Agreement, config Config, guildID string) ([]messages.Definition, error) {
	listChildren := []component{{Type: 10, Content: "# 🤝 Convenios"},
		{Type: 10, Content: fmt.Sprintf("**Convenios activos:** %d", len(items))},
		{Type: 14, Divider: true, Spacing: 1}}
	if len(items) == 0 {
		listChildren = append(listChildren, component{Type: 10, Content: "Aún no hay convenios registrados."})
	}
	for _, item := range items[:min(len(items), 10)] {
		content := fmt.Sprintf("## `%s`\n%s", item.ID, item.Description)
		if item.ImageURL == "" {
			listChildren = append(listChildren, component{Type: 10, Content: content})
		} else {
			listChildren = append(listChildren, component{Type: 9,
				Components: []component{{Type: 10, Content: content}},
				Accessory:  &component{Type: 11, Media: &media{URL: item.ImageURL}}})
		}
	}
	if len(items) > 10 {
		listChildren = append(listChildren, component{Type: 10,
			Content: fmt.Sprintf("Y **%d** convenios más.", len(items)-10)})
	}
	controlChildren := []component{{Type: 10, Content: "# 🤝 Administración de convenios"},
		{Type: 10, Content: "Añade un convenio al listado público. La imagen es opcional."},
		{Type: 14, Divider: true, Spacing: 1}, {Type: 1, Components: []component{
			{Type: 2, Style: 3, Label: "Añadir convenio", CustomID: ButtonAddCustomID},
			{Type: 2, Style: 1, Label: "Ver convenios", CustomID: ButtonListCustomID},
		}}}
	list, err := definition(agreementsListMessageKey, config.ChannelID, guildID, listChildren)
	if err != nil {
		return nil, err
	}
	control, err := definition(agreementsControlMessageKey, config.ControlChannelID, guildID, controlChildren)
	if err != nil {
		return nil, err
	}
	return []messages.Definition{list, control}, nil
}

func definition(key string, channelID string, guildID string, children []component) (messages.Definition, error) {
	encoded, err := json.Marshal(component{Type: 17, Accent: agreementsAccent, Components: children})
	if err != nil {
		return messages.Definition{}, fmt.Errorf("encode agreement panel: %w", err)
	}
	return messages.Definition{Key: key, GuildID: guildID, ChannelID: channelID,
		Payload: messages.Payload{Components: []messages.Component{encoded},
			AllowedMentions: messages.AllowedMentions{Parse: []string{}}}}, nil
}
