package agreementdiscord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const companyPageSize = 25
const agreementPageSize = 3

func (handler *handler) showCompanies(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, mode companyMode, page int) {
	items, err := handler.service.ListCompanies(ctx)
	if err != nil {
		handler.log.Error("list agreement companies", zap.Error(err))
		handler.edit(ctx, session, interaction, "No fue posible consultar las empresas.", nil)
		return
	}
	if len(items) == 0 {
		handler.edit(ctx, session, interaction, "Primero añade una empresa con el botón correspondiente.", nil)
		return
	}
	page = min(page, (len(items)-1)/companyPageSize)
	start, end := page*companyPageSize, min((page+1)*companyPageSize, len(items))
	options := make([]discordgo.SelectMenuOption, 0, end-start)
	for _, item := range items[start:end] {
		options = append(options, discordgo.SelectMenuOption{Label: item.Name, Value: item.ID, Description: item.ID})
	}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{CustomID: mode.selectID, Placeholder: mode.placeholder, Options: options, MaxValues: 1},
	}}}
	components = append(components, pageButtons(mode.pagePrefix, page, (len(items)-1)/companyPageSize)...)
	handler.edit(ctx, session, interaction, fmt.Sprintf("**%s** · Página %d", mode.title, page+1), components)
}

func (handler *handler) showList(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, page int) {
	items, err := handler.service.List(ctx)
	if err != nil {
		handler.log.Error("list agreements", zap.Error(err))
		handler.edit(ctx, session, interaction, "No fue posible consultar los convenios.", nil)
		return
	}
	if len(items) == 0 {
		handler.edit(ctx, session, interaction, "No hay convenios registrados.", nil)
		return
	}
	last := (len(items) - 1) / agreementPageSize
	page = min(page, last)
	var content strings.Builder
	fmt.Fprintf(&content, "**Convenios** · %d registros · Página %d/%d\n", len(items), page+1, last+1)
	for _, item := range items[page*agreementPageSize : min((page+1)*agreementPageSize, len(items))] {
		description := []rune(item.Description)
		if len(description) > 300 {
			description = append(description[:297], '.', '.', '.')
		}
		fmt.Fprintf(&content, "\n**%s** · `%s`\n%s\n", item.CompanyName, item.ID, string(description))
	}
	handler.edit(ctx, session, interaction, content.String(), pageButtons(listPagePrefix, page, last))
}

func pageButtons(prefix string, page, last int) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent
	if page > 0 {
		buttons = append(buttons, discordgo.Button{CustomID: fmt.Sprintf("%s%d", prefix, page-1), Label: "Anterior", Style: discordgo.SecondaryButton})
	}
	if page < last {
		buttons = append(buttons, discordgo.Button{CustomID: fmt.Sprintf("%s%d", prefix, page+1), Label: "Siguiente", Style: discordgo.SecondaryButton})
	}
	if len(buttons) == 0 {
		return nil
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

func (handler *handler) openModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, companyID string) {
	if len(companyID) > 64 {
		return
	}
	fields := []discordgo.TextInput{
		{CustomID: idInputID, Label: "Identificador del convenio", Style: discordgo.TextInputShort, Required: true, MaxLength: 64},
		{CustomID: descriptionInputID, Label: "Descripción", Style: discordgo.TextInputParagraph, Required: true, MinLength: 3, MaxLength: 1000},
		{CustomID: imageInputID, Label: "Imagen HTTPS (opcional)", Style: discordgo.TextInputShort, MaxLength: 400},
	}
	components := make([]discordgo.MessageComponent, 0, len(fields))
	for _, field := range fields {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{field}})
	}
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: addModalPrefix + companyID, Title: "Añadir convenio", Components: components},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("open agreement modal", zap.Error(err))
	}
}

func input(data discordgo.ModalSubmitInteractionData, id string) string {
	for _, component := range data.Components {
		var children []discordgo.MessageComponent
		switch row := component.(type) {
		case *discordgo.ActionsRow:
			children = row.Components
		case discordgo.ActionsRow:
			children = row.Components
		}
		for _, child := range children {
			switch field := child.(type) {
			case *discordgo.TextInput:
				if field.CustomID == id {
					return field.Value
				}
			case discordgo.TextInput:
				if field.CustomID == id {
					return field.Value
				}
			}
		}
	}
	return ""
}
