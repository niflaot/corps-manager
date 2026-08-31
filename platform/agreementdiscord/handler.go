package agreementdiscord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/agreements"
	"go.uber.org/zap"
)

const (
	interactionTimeout = 10 * time.Second
	addModalID         = "agreements:add:submit"
	idInputID          = "agreement_id"
	descriptionInputID = "agreement_description"
	imageInputID       = "agreement_image"
)

type handler struct {
	service *agreements.Service
	config  agreements.Config
	log     *zap.Logger
}

func (handler *handler) handle(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if !handler.config.Enabled || event.GuildID == "" || event.ChannelID != handler.config.ControlChannelID {
		return
	}
	switch customID(event) {
	case agreements.ButtonAddCustomID:
		handler.openModal(session, event.Interaction)
	case agreements.ButtonListCustomID:
		handler.showList(session, event)
	case addModalID:
		handler.submit(session, event)
	}
}

func (handler *handler) openModal(session *discordgo.Session, interaction *discordgo.Interaction) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{
			CustomID: idInputID, Label: "ID del convenio", Style: discordgo.TextInputShort,
			Placeholder: "Ejemplo: lspd", Required: true, MinLength: 1, MaxLength: 64}}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{
			CustomID: descriptionInputID, Label: "Descripción", Style: discordgo.TextInputParagraph,
			Placeholder: "Beneficios y condiciones del convenio", Required: true, MinLength: 3, MaxLength: 1000}}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{
			CustomID: imageInputID, Label: "URL HTTPS de la foto (opcional)", Style: discordgo.TextInputShort,
			Placeholder: "https://...", Required: false, MaxLength: 400}}},
	}
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: addModalID, Title: "Añadir convenio", Components: components},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("open agreement modal", zap.Error(err))
	}
}

func (handler *handler) submit(session *discordgo.Session, event *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	actor := interactionUserID(event)
	item, err := handler.service.Create(ctx, input(event.ModalSubmitData(), idInputID),
		input(event.ModalSubmitData(), descriptionInputID), input(event.ModalSubmitData(), imageInputID), actor)
	if err != nil {
		message := "❌ No fue posible añadir el convenio."
		switch {
		case errors.Is(err, agreements.ErrAlreadyExists):
			message = "⚠️ Ya existe un convenio con ese ID."
		case errors.Is(err, agreements.ErrInvalidID):
			message = "❌ El ID debe usar letras minúsculas, números, `_` o `-`."
		case errors.Is(err, agreements.ErrInvalidDescription):
			message = "❌ La descripción debe tener entre 3 y 1000 caracteres."
		case errors.Is(err, agreements.ErrInvalidImageURL):
			message = "❌ La foto debe ser una URL HTTPS válida o quedar vacía."
		default:
			handler.log.Error("create agreement", zap.Error(err))
		}
		handler.respond(ctx, session, event.Interaction, message)
		return
	}
	handler.respond(ctx, session, event.Interaction, fmt.Sprintf("✅ Convenio `%s` añadido al listado.", item.ID))
}

func (handler *handler) showList(session *discordgo.Session, event *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	items, err := handler.service.List(ctx)
	if err != nil {
		handler.log.Error("list agreements", zap.Error(err))
		handler.respond(ctx, session, event.Interaction, "❌ No fue posible consultar los convenios.")
		return
	}
	var content strings.Builder
	fmt.Fprintf(&content, "**Convenios activos** · Total: %d\n", len(items))
	if len(items) == 0 {
		content.WriteString("No hay convenios registrados.")
	} else {
		for _, item := range items[:min(len(items), 20)] {
			description := []rune(strings.ReplaceAll(item.Description, "\n", " "))
			if len(description) > 60 {
				description = append(description[:57], '.', '.', '.')
			}
			fmt.Fprintf(&content, "\n• `%s` — %s", item.ID, string(description))
		}
		if len(items) > 20 {
			fmt.Fprintf(&content, "\nY %d convenios más.", len(items)-20)
		}
	}
	handler.respond(ctx, session, event.Interaction, content.String())
}

func (handler *handler) respond(ctx context.Context, session *discordgo.Session,
	interaction *discordgo.Interaction, message string) {
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: message, Flags: discordgo.MessageFlagsEphemeral,
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("respond to agreement interaction", zap.Error(err))
	}
}

func customID(event *discordgo.InteractionCreate) string {
	if event.Type == discordgo.InteractionMessageComponent {
		return event.MessageComponentData().CustomID
	}
	if event.Type == discordgo.InteractionModalSubmit {
		return event.ModalSubmitData().CustomID
	}
	return ""
}

func interactionUserID(event *discordgo.InteractionCreate) string {
	if event.Member != nil && event.Member.User != nil {
		return event.Member.User.ID
	}
	if event.User != nil {
		return event.User.ID
	}
	return ""
}

func input(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, component := range data.Components {
		var row *discordgo.ActionsRow
		switch value := component.(type) {
		case *discordgo.ActionsRow:
			row = value
		case discordgo.ActionsRow:
			row = &value
		}
		if row == nil {
			continue
		}
		for _, child := range row.Components {
			switch field := child.(type) {
			case *discordgo.TextInput:
				if field.CustomID == customID {
					return field.Value
				}
			case discordgo.TextInput:
				if field.CustomID == customID {
					return field.Value
				}
			}
		}
	}
	return ""
}
