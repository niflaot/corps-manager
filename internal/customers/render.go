package customers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	customerMessageKey = "frequent-customers"
	customerAccent     = 0x3498db
	// ButtonRecordCustomID identifies the record-visit action.
	ButtonRecordCustomID = "customers:record"
	// ButtonDetailCustomID identifies the customer-detail action.
	ButtonDetailCustomID = "customers:detail"
	// ButtonDeleteCustomID identifies the owner-only delete action.
	ButtonDeleteCustomID = "customers:delete"
)

type component struct {
	Type       int         `json:"type"`
	Content    string      `json:"content,omitempty"`
	Divider    bool        `json:"divider,omitempty"`
	Spacing    int         `json:"spacing,omitempty"`
	Style      int         `json:"style,omitempty"`
	Label      string      `json:"label,omitempty"`
	CustomID   string      `json:"custom_id,omitempty"`
	URL        string      `json:"url,omitempty"`
	Components []component `json:"components,omitempty"`
	Accent     int         `json:"accent_color,omitempty"`
}

// Render creates the managed frequent-customer panel definition.
func Render(customers []Customer, config Config, guildID string) (messages.Definition, error) {
	var ranking strings.Builder
	ranking.WriteString("**Clientes con más visitas**\n")
	if len(customers) == 0 {
		ranking.WriteString("Aún no hay visitas registradas.")
	} else {
		for index, customer := range customers[:min(len(customers), 10)] {
			fmt.Fprintf(&ranking, "%d. `%s` — **%d** visitas · **$%d**\n",
				index+1, customer.Name, customer.Visits, customer.TotalSpent)
		}
		if len(customers) > 10 {
			fmt.Fprintf(&ranking, "\nY %d clientes más. Usa **Ver clientes**.", len(customers)-10)
		}
	}
	children := []component{{Type: 10, Content: "# ⭐ Clientes frecuentes"},
		{Type: 10, Content: ranking.String()}, {Type: 14, Divider: true, Spacing: 1},
		{Type: 1, Components: []component{
			{Type: 2, Style: 3, Label: "Registrar atención", CustomID: ButtonRecordCustomID},
			{Type: 2, Style: 5, Label: "Ver clientes", URL: config.PublicURL},
			{Type: 2, Style: 2, Label: "Consultar cliente", CustomID: ButtonDetailCustomID},
			{Type: 2, Style: 4, Label: "Eliminar cliente", CustomID: ButtonDeleteCustomID},
		}}}
	encoded, err := json.Marshal(component{Type: 17, Accent: customerAccent, Components: children})
	if err != nil {
		return messages.Definition{}, fmt.Errorf("encode customer panel: %w", err)
	}
	return messages.Definition{Key: customerMessageKey, GuildID: guildID, ChannelID: config.ChannelID,
		Payload: messages.Payload{Components: []messages.Component{encoded},
			AllowedMentions: messages.AllowedMentions{Parse: []string{}}}}, nil
}
