package discord

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/inactivity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	inactivityAddModalID    = "inactivity:add:submit"
	inactivityRemoveModalID = "inactivity:remove:submit"
	inactivityNameInputID   = "employee_name"
	interactionTimeout      = 10 * time.Second
)

type inactivityInteractionHandler struct {
	service *inactivity.Service
	config  inactivity.Config
	log     *zap.Logger
}

func registerInactivityInteractions(lifecycle fx.Lifecycle, client *Client, service *inactivity.Service,
	config inactivity.Config, log *zap.Logger) {
	handler := &inactivityInteractionHandler{service: service, config: config, log: log}
	remove := client.AddHandler(handler.handle)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		remove()
		return nil
	}})
}

func (handler *inactivityInteractionHandler) handle(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if !handler.config.Enabled || event.GuildID == "" || event.ChannelID != handler.config.ChannelID {
		return
	}
	customID := interactionCustomID(event)
	if customID == "" {
		return
	}
	if page, update, ok := inactivityListPage(customID); ok {
		handler.showInactivityList(session, event, page, update)
		return
	}
	if customID != inactivity.ButtonAddCustomID && customID != inactivity.ButtonRemoveCustomID &&
		customID != inactivityAddModalID && customID != inactivityRemoveModalID {
		return
	}
	if !canManageRegistry(event) {
		handler.respond(session, event.Interaction, "No tienes permiso para administrar este registro.")
		return
	}
	switch customID {
	case inactivity.ButtonAddCustomID:
		handler.openModal(session, event.Interaction, inactivityAddModalID, "Añadir empleado expulsado")
	case inactivity.ButtonRemoveCustomID:
		handler.openModal(session, event.Interaction, inactivityRemoveModalID, "Retirar empleado del registro")
	case inactivityAddModalID, inactivityRemoveModalID:
		handler.submit(session, event, customID)
	}
}

func (handler *inactivityInteractionHandler) openModal(session *discordgo.Session, interaction *discordgo.Interaction,
	customID string, title string) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: customID, Title: title,
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: inactivityNameInputID, Label: "Nombre_Apellido",
					Style: discordgo.TextInputShort, Placeholder: "Ejemplo: Thomas_Jhonson", Required: true,
					MinLength: 3, MaxLength: 64},
			}}},
		},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("open inactivity registry modal", zap.Error(err))
	}
}

func (handler *inactivityInteractionHandler) submit(session *discordgo.Session, event *discordgo.InteractionCreate,
	customID string) {
	name := modalInput(event.ModalSubmitData(), inactivityNameInputID)
	actor := ""
	if event.Member != nil && event.Member.User != nil {
		actor = event.Member.User.ID
	}
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	var err error
	message := ""
	if customID == inactivityAddModalID {
		_, err = handler.service.Add(ctx, name, actor)
		message = fmt.Sprintf("✅ `%s` fue añadido al registro.", name)
	} else {
		err = handler.service.Remove(ctx, name)
		message = fmt.Sprintf("✅ `%s` fue retirado del registro.", name)
	}
	if err != nil {
		message = inactivityInteractionError(err)
		if !errors.Is(err, inactivity.ErrInvalidName) && !errors.Is(err, inactivity.ErrAlreadyExists) &&
			!errors.Is(err, inactivity.ErrNotFound) {
			handler.log.Error("update inactivity registry", zap.String("employee", name), zap.Error(err))
		}
	}
	handler.respondWithContext(ctx, session, event.Interaction, message)
}

func (handler *inactivityInteractionHandler) respond(session *discordgo.Session, interaction *discordgo.Interaction,
	message string) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	handler.respondWithContext(ctx, session, interaction, message)
}

func (handler *inactivityInteractionHandler) respondWithContext(ctx context.Context, session *discordgo.Session,
	interaction *discordgo.Interaction, message string) {
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: message, Flags: discordgo.MessageFlagsEphemeral,
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("respond to inactivity registry interaction", zap.Error(err))
	}
}

func interactionCustomID(event *discordgo.InteractionCreate) string {
	switch event.Type {
	case discordgo.InteractionMessageComponent:
		return event.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		return event.ModalSubmitData().CustomID
	default:
		return ""
	}
}

func canManageRegistry(event *discordgo.InteractionCreate) bool {
	if event.Member == nil {
		return false
	}
	permissions := event.Member.Permissions
	return permissions&discordgo.PermissionAdministrator != 0 || permissions&discordgo.PermissionManageMessages != 0
}

func modalInput(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, rowComponent := range data.Components {
		row, ok := rowComponent.(*discordgo.ActionsRow)
		if !ok {
			if value, valueOK := rowComponent.(discordgo.ActionsRow); valueOK {
				row = &value
			}
		}
		if row == nil {
			continue
		}
		for _, inputComponent := range row.Components {
			switch input := inputComponent.(type) {
			case *discordgo.TextInput:
				if input.CustomID == customID {
					return input.Value
				}
			case discordgo.TextInput:
				if input.CustomID == customID {
					return input.Value
				}
			}
		}
	}
	return ""
}

func inactivityInteractionError(err error) string {
	switch {
	case errors.Is(err, inactivity.ErrInvalidName):
		return "❌ Usa exactamente el formato `Nombre_Apellido`, únicamente con letras."
	case errors.Is(err, inactivity.ErrAlreadyExists):
		return "⚠️ Ese empleado ya está registrado."
	case errors.Is(err, inactivity.ErrNotFound):
		return "⚠️ Ese empleado no aparece en el registro."
	default:
		return "❌ No fue posible actualizar el registro. Revisa los logs del bot."
	}
}
