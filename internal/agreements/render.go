package agreements

import (
	"encoding/json"
	"fmt"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	legacyListMessageKey        = "business-agreements"
	companyMessageKeyPrefix     = "agr-"
	maximumPublicAgreements     = 3
	agreementsControlMessageKey = "business-agreements-control"
	agreementsAccent            = 0x9b59b6
	// ButtonAddCustomID identifies the add-agreement action.
	ButtonAddCustomID = "agreements:add"
	// ButtonCompanyAddCustomID identifies the add-company action.
	ButtonCompanyAddCustomID = "agreements:company-add"
	// ButtonCompanyEditCustomID identifies the edit-company action.
	ButtonCompanyEditCustomID = "agreements:company-edit"
	// ButtonCompanyDeleteCustomID identifies the delete-company action.
	ButtonCompanyDeleteCustomID = "agreements:company-delete"
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

// Render creates one public agreement list per company with a channel and the shared control panel.
func Render(companies []Company, items []Agreement, config Config, guildID string) ([]messages.Definition, error) {
	byCompany := make(map[string][]Agreement, len(companies))
	for _, item := range items {
		byCompany[item.CompanyID] = append(byCompany[item.CompanyID], item)
	}
	definitions := make([]messages.Definition, 0, len(companies)+1)
	for _, company := range companies {
		if company.ChannelID == "" || len(company.ID) > maximumCompanyIDLength {
			continue
		}
		list, err := definition(companyMessageKey(company.ID), company.ChannelID, guildID,
			listChildren(company, byCompany[company.ID]))
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, list)
	}
	controlChildren := []component{{Type: 10, Content: "# 🤝 Administración de convenios"},
		{Type: 10, Content: "Gestiona las empresas, el canal de cada una y sus convenios. La imagen es opcional."},
		{Type: 14, Divider: true, Spacing: 1}, {Type: 1, Components: []component{
			{Type: 2, Style: 3, Label: "Añadir convenio", CustomID: ButtonAddCustomID},
			{Type: 2, Style: 1, Label: "Ver convenios", CustomID: ButtonListCustomID},
		}}, {Type: 1, Components: []component{
			{Type: 2, Style: 3, Label: "Añadir empresa", CustomID: ButtonCompanyAddCustomID},
			{Type: 2, Style: 2, Label: "Editar empresa", CustomID: ButtonCompanyEditCustomID},
			{Type: 2, Style: 4, Label: "Eliminar empresa", CustomID: ButtonCompanyDeleteCustomID},
		}}}
	control, err := definition(agreementsControlMessageKey, config.ControlChannelID, guildID, controlChildren)
	if err != nil {
		return nil, err
	}
	return append(definitions, control), nil
}

func companyMessageKey(companyID string) string { return companyMessageKeyPrefix + companyID }

func listChildren(company Company, items []Agreement) []component {
	children := []component{{Type: 10, Content: "# 🤝 Convenios · " + company.Name},
		{Type: 10, Content: fmt.Sprintf("**Convenios activos:** %d", len(items))},
		{Type: 14, Divider: true, Spacing: 1}}
	if len(items) == 0 {
		children = append(children, component{Type: 10, Content: "Aún no hay convenios registrados."})
	}
	for _, item := range items[:min(len(items), maximumPublicAgreements)] {
		content := fmt.Sprintf("## `%s`\n%s", item.ID, item.Description)
		if item.ImageURL == "" {
			children = append(children, component{Type: 10, Content: content})
		} else {
			children = append(children, component{Type: 9, Components: []component{{Type: 10, Content: content}},
				Accessory: &component{Type: 11, Media: &media{URL: item.ImageURL}}})
		}
	}
	if len(items) > maximumPublicAgreements {
		children = append(children, component{Type: 10,
			Content: fmt.Sprintf("Y **%d** convenios más.", len(items)-maximumPublicAgreements)})
	}
	return children
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
