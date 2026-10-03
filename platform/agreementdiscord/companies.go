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
	companyCreateModalID      = "agreements:company-submit"
	companyUpdateModalPrefix  = "agreements:company-update:"
	companyEditSelectID       = "agreements:company-edit-select"
	companyDeleteSelectID     = "agreements:company-delete-select"
	companyEditPagePrefix     = "agreements:companies-edit:"
	companyDeletePagePrefix   = "agreements:companies-delete:"
	companyIDInputID          = "company_id"
	companyNameInputID        = "company_name"
	companyChannelInputID     = "company_channel"
	maximumCompanyIDInputSize = 60
)

type companyMode struct {
	selectID    string
	pagePrefix  string
	placeholder string
	title       string
}

var (
	agreementMode = companyMode{companySelectID, companyPagePrefix, "Elige la empresa del convenio", "Empresa del convenio"}
	editMode      = companyMode{companyEditSelectID, companyEditPagePrefix, "Elige la empresa a editar", "Editar empresa"}
	deleteMode    = companyMode{companyDeleteSelectID, companyDeletePagePrefix, "Elige la empresa a eliminar", "Eliminar empresa"}
)

func (handler *handler) handleCompany(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, id string) bool {
	interaction := event.Interaction
	switch event.Type {
	case discordgo.InteractionModalSubmit:
		switch {
		case id == companyCreateModalID:
			if handler.deferResponse(ctx, session, interaction) {
				handler.saveCompany(ctx, session, event, "")
			}
		case strings.HasPrefix(id, companyUpdateModalPrefix):
			if handler.deferResponse(ctx, session, interaction) {
				handler.saveCompany(ctx, session, event, strings.TrimPrefix(id, companyUpdateModalPrefix))
			}
		default:
			return false
		}
	case discordgo.InteractionMessageComponent:
		return handler.handleCompanyComponent(ctx, session, event, id)
	default:
		return false
	}
	return true
}

func (handler *handler) handleCompanyComponent(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, id string) bool {
	interaction := event.Interaction
	values := event.MessageComponentData().Values
	switch {
	case id == agreements.ButtonCompanyAddCustomID:
		handler.openCompanyModal(ctx, session, interaction, agreements.Company{})
	case id == agreements.ButtonCompanyEditCustomID:
		handler.showModeCompanies(ctx, session, interaction, editMode, 0)
	case id == agreements.ButtonCompanyDeleteCustomID:
		handler.showModeCompanies(ctx, session, interaction, deleteMode, 0)
	case strings.HasPrefix(id, companyEditPagePrefix):
		handler.showModePage(ctx, session, interaction, editMode, strings.TrimPrefix(id, companyEditPagePrefix))
	case strings.HasPrefix(id, companyDeletePagePrefix):
		handler.showModePage(ctx, session, interaction, deleteMode, strings.TrimPrefix(id, companyDeletePagePrefix))
	case id == companyEditSelectID && len(values) == 1:
		handler.editCompany(ctx, session, interaction, values[0])
	case id == companyDeleteSelectID && len(values) == 1:
		if handler.deferResponse(ctx, session, interaction) {
			handler.removeCompany(ctx, session, interaction, values[0])
		}
	default:
		return false
	}
	return true
}

func (handler *handler) showModeCompanies(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, mode companyMode, page int) {
	if handler.deferResponse(ctx, session, interaction) {
		handler.showCompanies(ctx, session, interaction, mode, page)
	}
}

func (handler *handler) showModePage(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, mode companyMode, value string) {
	if page, err := strconv.Atoi(value); err == nil && page >= 0 {
		handler.showModeCompanies(ctx, session, interaction, mode, page)
	}
}

func (handler *handler) editCompany(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, id string) {
	items, err := handler.service.ListCompanies(ctx)
	if err != nil {
		handler.log.Error("list agreement companies", zap.Error(err))
		return
	}
	for _, item := range items {
		if item.ID == id {
			handler.openCompanyModal(ctx, session, interaction, item)
			return
		}
	}
}

func (handler *handler) openCompanyModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, company agreements.Company) {
	customID, title := companyCreateModalID, "Añadir empresa"
	fields := []discordgo.TextInput{
		{CustomID: companyIDInputID, Label: "Identificador de la empresa", Style: discordgo.TextInputShort, Required: true, MaxLength: maximumCompanyIDInputSize},
		{CustomID: companyNameInputID, Label: "Nombre", Style: discordgo.TextInputShort, Required: true, MinLength: 2, MaxLength: 80, Value: company.Name},
		{CustomID: companyChannelInputID, Label: "ID del canal de convenios", Style: discordgo.TextInputShort, Required: true, MaxLength: 20, Value: company.ChannelID},
	}
	if company.ID != "" {
		customID, title, fields = companyUpdateModalPrefix+company.ID, "Editar empresa", fields[1:]
	}
	components := make([]discordgo.MessageComponent, 0, len(fields))
	for _, field := range fields {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{field}})
	}
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: customID, Title: title, Components: components},
	}, discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("open company modal", zap.Error(err))
	}
}

func (handler *handler) saveCompany(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, id string) {
	data := event.ModalSubmitData()
	var company agreements.Company
	var err error
	if id == "" {
		company, err = handler.service.CreateCompany(ctx, input(data, companyIDInputID), input(data, companyNameInputID), input(data, companyChannelInputID))
	} else {
		company, err = handler.service.UpdateCompany(ctx, id, input(data, companyNameInputID), input(data, companyChannelInputID))
	}
	content := fmt.Sprintf("✅ Empresa **%s** guardada; sus convenios se publican en <#%s>.", company.Name, company.ChannelID)
	if err != nil {
		content = handler.companyFailure(err)
	}
	handler.edit(ctx, session, event.Interaction, content, nil)
}

func (handler *handler) removeCompany(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, id string) {
	content := fmt.Sprintf("✅ Empresa `%s` eliminada.", id)
	if err := handler.service.DeleteCompany(ctx, id); err != nil {
		content = handler.companyFailure(err)
	}
	handler.edit(ctx, session, interaction, content, nil)
}

func (handler *handler) companyFailure(err error) string {
	switch {
	case errors.Is(err, agreements.ErrAlreadyExists):
		return "Ya existe una empresa con ese identificador."
	case errors.Is(err, agreements.ErrInvalidCompany):
		return "Revisa los datos: identificador en minúsculas (máx. 60), nombre de 2 a 80 caracteres y un ID de canal válido."
	case errors.Is(err, agreements.ErrCompanyNotFound):
		return "La empresa ya no existe."
	case errors.Is(err, agreements.ErrCompanyInUse):
		return "La empresa todavía tiene convenios; no se puede eliminar."
	}
	handler.log.Error("manage agreement company", zap.Error(err))
	return "No fue posible completar la operación. Consulta las empresas antes de reintentar."
}
