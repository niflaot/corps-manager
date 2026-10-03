package agreementdiscord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/agreements"
	"go.uber.org/zap"
)

const (
	interactionTimeout = 15 * time.Second
	addModalPrefix     = "agreements:submit:"
	companySelectID    = "agreements:company"
	companyPagePrefix  = "agreements:companies:"
	listPagePrefix     = "agreements:page:"
	idInputID          = "agreement_id"
	descriptionInputID = "agreement_description"
	imageInputID       = "agreement_image"
)

type handler struct {
	ctx     context.Context
	guildID string
	service *agreements.Service
	config  agreements.Config
	log     *zap.Logger
}

func (handler *handler) handle(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if !handler.config.Enabled || event.GuildID != handler.guildID || event.ChannelID != handler.config.ControlChannelID || event.Member == nil || event.Member.User == nil {
		return
	}
	id := customID(event)
	ctx, cancel := context.WithTimeout(handler.ctx, interactionTimeout)
	defer cancel()
	if handler.handleCompany(ctx, session, event, id) || handler.handleItem(ctx, session, event, id) {
		return
	}
	if event.Type == discordgo.InteractionMessageComponent && id == companySelectID {
		values := event.MessageComponentData().Values
		if len(values) == 1 {
			handler.openModal(ctx, session, event.Interaction, values[0])
		}
		return
	}
	if event.Type == discordgo.InteractionModalSubmit && strings.HasPrefix(id, addModalPrefix) {
		if handler.deferResponse(ctx, session, event.Interaction) {
			handler.submit(ctx, session, event, strings.TrimPrefix(id, addModalPrefix))
		}
		return
	}
	if event.Type != discordgo.InteractionMessageComponent {
		return
	}
	switch {
	case id == agreements.ButtonAddCustomID:
		if handler.deferResponse(ctx, session, event.Interaction) {
			handler.showCompanies(ctx, session, event.Interaction, agreementMode, 0)
		}
	case strings.HasPrefix(id, companyPagePrefix):
		page, err := strconv.Atoi(strings.TrimPrefix(id, companyPagePrefix))
		if err == nil && page >= 0 && handler.deferResponse(ctx, session, event.Interaction) {
			handler.showCompanies(ctx, session, event.Interaction, agreementMode, page)
		}
	case id == agreements.ButtonListCustomID:
		if handler.deferResponse(ctx, session, event.Interaction) {
			handler.showList(ctx, session, event.Interaction, 0)
		}
	case strings.HasPrefix(id, listPagePrefix):
		page, err := strconv.Atoi(strings.TrimPrefix(id, listPagePrefix))
		if err == nil && page >= 0 && handler.deferResponse(ctx, session, event.Interaction) {
			handler.showList(ctx, session, event.Interaction, page)
		}
	}
}

func (handler *handler) submit(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, companyID string) {
	item, err := handler.service.Create(ctx, companyID, input(event.ModalSubmitData(), idInputID),
		input(event.ModalSubmitData(), descriptionInputID), input(event.ModalSubmitData(), imageInputID), event.Member.User.ID)
	content := fmt.Sprintf("✅ Convenio `%s` añadido a %s.", item.ID, item.CompanyName)
	if err != nil {
		switch {
		case errors.Is(err, agreements.ErrAlreadyExists):
			content = "Ya existe ese identificador para la empresa seleccionada."
		case errors.Is(err, agreements.ErrCompanyNotFound), errors.Is(err, agreements.ErrInvalidCompany):
			content = "La empresa ya no está disponible. Abre el formulario nuevamente."
		case errors.Is(err, agreements.ErrInvalidID):
			content = "El identificador debe usar letras minúsculas, números, `_` o `-`."
		case errors.Is(err, agreements.ErrInvalidDescription):
			content = "La descripción debe tener entre 3 y 1000 caracteres."
		case errors.Is(err, agreements.ErrInvalidImageURL):
			content = "La imagen debe ser una URL HTTPS válida o quedar vacía."
		default:
			content = "No fue posible completar la operación. Consulta el listado antes de reintentar."
			handler.log.Error("create agreement", zap.Error(err))
		}
	}
	handler.edit(ctx, session, event.Interaction, content, nil)
}

func (handler *handler) deferResponse(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction) bool {
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("acknowledge agreement interaction", zap.Error(err))
	}
	return err == nil
}

func (handler *handler) edit(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, content string, components []discordgo.MessageComponent) {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	_, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{Content: &content, Components: &components,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("respond to agreement interaction", zap.Error(err))
	}
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
