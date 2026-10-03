package agreementdiscord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/agreements"
	"go.uber.org/zap"
)

const (
	itemUpdateModalPrefix     = "agreements:item-update:"
	itemEditCompanySelectID   = "agreements:item-edit-company"
	itemEditCompaniesPrefix   = "agreements:item-edit-companies:"
	itemEditSelectPrefix      = "agreements:item-edit:"
	itemEditPagePrefix        = "agreements:items-edit:"
	itemDeleteCompanyID       = "agreements:item-delete-company"
	itemDeleteCompaniesPrefix = "agreements:item-delete-companies:"
	itemDeleteSelectPrefix    = "agreements:item-delete:"
	itemDeletePagePrefix      = "agreements:items-delete:"
	maximumCustomIDLength     = 100
	maximumOptionDescription  = 100
)

type itemMode struct {
	button       string
	companies    companyMode
	selectPrefix string
	pagePrefix   string
	placeholder  string
}

var (
	itemEditMode = itemMode{agreements.ButtonItemEditCustomID,
		companyMode{itemEditCompanySelectID, itemEditCompaniesPrefix, "Elige la empresa del convenio", "Editar convenio"},
		itemEditSelectPrefix, itemEditPagePrefix, "Elige el convenio a editar"}
	itemDeleteMode = itemMode{agreements.ButtonItemDeleteCustomID,
		companyMode{itemDeleteCompanyID, itemDeleteCompaniesPrefix, "Elige la empresa del convenio", "Eliminar convenio"},
		itemDeleteSelectPrefix, itemDeletePagePrefix, "Elige el convenio a eliminar"}
)

func (handler *handler) handleItem(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, id string) bool {
	interaction := event.Interaction
	if event.Type == discordgo.InteractionModalSubmit {
		companyID, itemID, found := strings.Cut(strings.TrimPrefix(id, itemUpdateModalPrefix), ":")
		if !strings.HasPrefix(id, itemUpdateModalPrefix) || !found {
			return false
		}
		if handler.deferResponse(ctx, session, interaction) {
			handler.saveItem(ctx, session, event, companyID, itemID)
		}
		return true
	}
	if event.Type != discordgo.InteractionMessageComponent {
		return false
	}
	values := event.MessageComponentData().Values
	for _, mode := range []itemMode{itemEditMode, itemDeleteMode} {
		switch {
		case id == mode.button:
			handler.showModeCompanies(ctx, session, interaction, mode.companies, 0)
		case strings.HasPrefix(id, mode.companies.pagePrefix):
			handler.showModePage(ctx, session, interaction, mode.companies, strings.TrimPrefix(id, mode.companies.pagePrefix))
		case id == mode.companies.selectID && len(values) == 1:
			if handler.deferResponse(ctx, session, interaction) {
				handler.showItems(ctx, session, interaction, mode, values[0], 0)
			}
		case strings.HasPrefix(id, mode.pagePrefix):
			companyID, value, _ := strings.Cut(strings.TrimPrefix(id, mode.pagePrefix), ":")
			if page, err := strconv.Atoi(value); err == nil && page >= 0 && handler.deferResponse(ctx, session, interaction) {
				handler.showItems(ctx, session, interaction, mode, companyID, page)
			}
		case strings.HasPrefix(id, mode.selectPrefix) && len(values) == 1:
			handler.chooseItem(ctx, session, interaction, mode, strings.TrimPrefix(id, mode.selectPrefix), values[0])
		default:
			continue
		}
		return true
	}
	return false
}

func (handler *handler) companyItems(ctx context.Context, companyID string) ([]agreements.Agreement, error) {
	items, err := handler.service.List(ctx)
	selected := make([]agreements.Agreement, 0, len(items))
	for _, item := range items {
		if item.CompanyID == companyID {
			selected = append(selected, item)
		}
	}
	return selected, err
}

func (handler *handler) showItems(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, mode itemMode, companyID string, page int) {
	items, err := handler.companyItems(ctx, companyID)
	if err != nil {
		handler.log.Error("list company agreements", zap.Error(err))
		handler.edit(ctx, session, interaction, "No fue posible consultar los convenios.", nil)
		return
	}
	if len(items) == 0 {
		handler.edit(ctx, session, interaction, "Esa empresa no tiene convenios registrados.", nil)
		return
	}
	last := (len(items) - 1) / companyPageSize
	page = min(page, last)
	options := make([]discordgo.SelectMenuOption, 0, companyPageSize)
	for _, item := range items[page*companyPageSize : min((page+1)*companyPageSize, len(items))] {
		options = append(options, discordgo.SelectMenuOption{Label: item.ID, Value: item.ID, Description: shorten(item.Description, maximumOptionDescription)})
	}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{CustomID: mode.selectPrefix + companyID, Placeholder: mode.placeholder, Options: options, MaxValues: 1},
	}}}
	components = append(components, pageButtons(mode.pagePrefix+companyID+":", page, last)...)
	handler.edit(ctx, session, interaction, fmt.Sprintf("**%s · %s** · Página %d", mode.companies.title, items[0].CompanyName, page+1), components)
}

func (handler *handler) chooseItem(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, mode itemMode, companyID, itemID string) {
	if mode.button == itemDeleteMode.button {
		if handler.deferResponse(ctx, session, interaction) {
			content := fmt.Sprintf("✅ Convenio `%s` eliminado.", itemID)
			if err := handler.service.DeleteAgreement(ctx, companyID, itemID); err != nil {
				content = handler.itemFailure(err)
			}
			handler.edit(ctx, session, interaction, content, nil)
		}
		return
	}
	modalID := itemUpdateModalPrefix + companyID + ":" + itemID
	items, err := handler.companyItems(ctx, companyID)
	if err != nil || len(modalID) > maximumCustomIDLength {
		handler.log.Error("prepare agreement edit", zap.Error(err), zap.Int("customIDLength", len(modalID)))
		if handler.deferResponse(ctx, session, interaction) {
			handler.edit(ctx, session, interaction, "Este convenio no se puede editar desde Discord: su identificador es demasiado largo.", nil)
		}
		return
	}
	for _, item := range items {
		if item.ID == itemID {
			handler.openItemModal(ctx, session, interaction, modalID, item)
			return
		}
	}
}

func (handler *handler) openItemModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, modalID string, item agreements.Agreement) {
	fields := []discordgo.TextInput{
		{CustomID: descriptionInputID, Label: "Descripción", Style: discordgo.TextInputParagraph, Required: true, MinLength: 3, MaxLength: 1000, Value: item.Description},
		{CustomID: imageInputID, Label: "Imagen HTTPS (opcional)", Style: discordgo.TextInputShort, MaxLength: 400, Value: item.ImageURL},
	}
	components := make([]discordgo.MessageComponent, 0, len(fields))
	for _, field := range fields {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{field}})
	}
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: modalID, Title: "Editar convenio " + shorten(item.ID, 30), Components: components},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("open agreement edit modal", zap.Error(err))
	}
}

func (handler *handler) saveItem(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, companyID, itemID string) {
	data := event.ModalSubmitData()
	content := fmt.Sprintf("✅ Convenio `%s` actualizado.", itemID)
	if _, err := handler.service.UpdateAgreement(ctx, companyID, itemID, input(data, descriptionInputID), input(data, imageInputID)); err != nil {
		content = handler.itemFailure(err)
	}
	handler.edit(ctx, session, event.Interaction, content, nil)
}

func (handler *handler) itemFailure(err error) string {
	switch {
	case errors.Is(err, agreements.ErrAgreementNotFound):
		return "El convenio ya no existe. Abre la lista nuevamente."
	case errors.Is(err, agreements.ErrInvalidDescription):
		return "La descripción debe tener entre 3 y 1000 caracteres."
	case errors.Is(err, agreements.ErrInvalidImageURL):
		return "La imagen debe ser una URL HTTPS válida o quedar vacía."
	}
	handler.log.Error("manage agreement", zap.Error(err))
	return "No fue posible completar la operación. Consulta el listado antes de reintentar."
}

func shorten(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(append(runes[:limit-3], '.', '.', '.'))
}
