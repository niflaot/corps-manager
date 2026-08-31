package customerdiscord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/customers"
	"go.uber.org/zap"
)

const (
	interactionTimeout = 10 * time.Second
	recordModalID      = "customers:record:submit"
	detailModalID      = "customers:detail:submit"
	deleteModalID      = "customers:delete:submit"
	nameInputID        = "customer_name"
)

type handler struct {
	service *customers.Service
	config  customers.Config
	log     *zap.Logger
}

func (handler *handler) handle(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if !handler.config.Enabled || event.GuildID == "" || event.ChannelID != handler.config.ChannelID {
		return
	}
	customID := customID(event)
	switch customID {
	case customers.ButtonRecordCustomID:
		handler.openNameModal(session, event.Interaction, recordModalID, "Registrar atención")
	case customers.ButtonListCustomID:
		handler.showList(session, event)
	case customers.ButtonDetailCustomID:
		handler.openNameModal(session, event.Interaction, detailModalID, "Consultar cliente")
	case customers.ButtonDeleteCustomID:
		if handler.requireOwner(session, event) {
			handler.openNameModal(session, event.Interaction, deleteModalID, "Eliminar cliente")
		}
	case recordModalID:
		handler.record(session, event)
	case detailModalID:
		handler.showDetail(session, event)
	case deleteModalID:
		if handler.requireOwner(session, event) {
			handler.delete(session, event)
		}
	}
}

func (handler *handler) openNameModal(session *discordgo.Session, interaction *discordgo.Interaction,
	modalID string, title string) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: modalID, Title: title,
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: nameInputID, Label: "Nombre del cliente", Style: discordgo.TextInputShort,
					Placeholder: "Ejemplo: Jane Doe", Required: true, MinLength: 2, MaxLength: 64},
			}}}},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("open customer modal", zap.Error(err))
	}
}

func (handler *handler) record(session *discordgo.Session, event *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	userID, displayName := actor(event)
	name := input(event.ModalSubmitData(), nameInputID)
	customer, err := handler.service.Record(ctx, name, userID, displayName)
	if err != nil {
		handler.respondError(ctx, session, event.Interaction, "registrar la atención", err)
		return
	}
	handler.respond(ctx, session, event.Interaction, fmt.Sprintf(
		"✅ Atención registrada para `%s`. Ahora tiene **%d** visitas.", customer.Name, customer.Visits))
}

func (handler *handler) delete(session *discordgo.Session, event *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	name := input(event.ModalSubmitData(), nameInputID)
	normalized, err := customers.NormalizeName(name)
	if err == nil {
		err = handler.service.Delete(ctx, name)
	}
	if err != nil {
		handler.respondError(ctx, session, event.Interaction, "eliminar el cliente", err)
		return
	}
	handler.respond(ctx, session, event.Interaction, "✅ Se eliminó `"+normalized+"` y todo su historial.")
}

func (handler *handler) requireOwner(session *discordgo.Session, event *discordgo.InteractionCreate) bool {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	guild, err := session.Guild(event.GuildID, discordgo.WithContext(ctx))
	userID, _ := actor(event)
	if err != nil {
		handler.log.Error("read Discord guild owner", zap.Error(err))
		handler.respond(ctx, session, event.Interaction, "❌ No fue posible validar al dueño del servidor.")
		return false
	}
	if guild.OwnerID != userID {
		handler.respond(ctx, session, event.Interaction, "⛔ Solo el dueño del servidor puede eliminar clientes.")
		return false
	}
	return true
}

func (handler *handler) respondError(ctx context.Context, session *discordgo.Session,
	interaction *discordgo.Interaction, operation string, err error) {
	message := "❌ No fue posible " + operation + "."
	switch {
	case errors.Is(err, customers.ErrInvalidName):
		message = "❌ Usa letras y números. El nombre se guardará en minúsculas y con `_` entre palabras."
	case errors.Is(err, customers.ErrNotFound):
		message = "⚠️ Ese cliente no existe."
	case errors.Is(err, customers.ErrInvalidAttendant):
		message = "❌ No fue posible identificar tu usuario de Discord."
	default:
		handler.log.Error("customer interaction failed", zap.String("operation", operation), zap.Error(err))
	}
	handler.respond(ctx, session, interaction, message)
}

func customID(event *discordgo.InteractionCreate) string {
	switch event.Type {
	case discordgo.InteractionMessageComponent:
		return event.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		return event.ModalSubmitData().CustomID
	default:
		return ""
	}
}

func actor(event *discordgo.InteractionCreate) (string, string) {
	user := event.User
	name := ""
	if event.Member != nil {
		name, user = strings.TrimSpace(event.Member.Nick), event.Member.User
	}
	if user == nil {
		return "", ""
	}
	if name == "" {
		name = strings.TrimSpace(user.GlobalName)
	}
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	return user.ID, name
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
