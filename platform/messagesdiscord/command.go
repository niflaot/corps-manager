package messagesdiscord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/zap"
)

const (
	commandName   = "mensaje"
	customPrefix  = "msg:"
	modalPrefix   = "msg:m:"
	confirmPrefix = "msg:ok:"
	cancelPrefix  = "msg:no:"
	noValue       = "-"
	subCreate     = "crear"
	subEdit       = "editar"
	subPreview    = "previsualizar"
	subShow       = "ver"
	subArchive    = "archivar"
	subList       = "listar"
	optionKey     = "clave"
	optionChannel = "canal"
	optionFile    = "archivo"
	// autocompleteLimit is Discord's maximum number of autocomplete choices.
	autocompleteLimit = 25
)

type command struct {
	manager Manager
	guildID string
	fetch   Fetcher
	pending *pendingStore
	log     *zap.Logger
}

func (c *command) definition() *discordgo.ApplicationCommand {
	key := func(complete bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optionKey, Description: "Clave del mensaje gestionado",
			Required: true, MaxLength: 64, Autocomplete: complete}
	}
	channel := func(required bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionChannel, Name: optionChannel, Description: "Canal donde se publica",
			Required: required, ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews}}
	}
	file := &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionAttachment, Name: optionFile, Description: "Archivo .json; si se omite se abre un formulario"}
	sub := func(name, description string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: description, Options: options}
	}
	return &discordgo.ApplicationCommand{Name: commandName, Description: "Gestiona mensajes estáticos con JSON de Components V2", Options: []*discordgo.ApplicationCommandOption{
		sub(subCreate, "Crea un mensaje desde JSON con vista previa", key(false), channel(true), file),
		sub(subEdit, "Reemplaza el JSON de un mensaje con vista previa", key(true), channel(false), file),
		sub(subPreview, "Valida un JSON y muestra su vista previa sin publicar", file),
		sub(subShow, "Exporta el JSON vigente de un mensaje", key(true)),
		sub(subArchive, "Deja de gestionar un mensaje", key(true)),
		sub(subList, "Lista los mensajes gestionados"),
	}}
}

func (c *command) handle(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	switch event.Type {
	case discordgo.InteractionApplicationCommand:
		c.run(ctx, session, event)
	case discordgo.InteractionApplicationCommandAutocomplete:
		c.autocomplete(ctx, session, event)
	case discordgo.InteractionMessageComponent:
		c.component(ctx, session, event)
	case discordgo.InteractionModalSubmit:
		c.modal(ctx, session, event)
	}
}

func (c *command) run(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	data := event.ApplicationCommandData()
	if len(data.Options) == 0 {
		return
	}
	sub := data.Options[0]
	item := pending{Key: stringOption(sub.Options, optionKey), ChannelID: stringOption(sub.Options, optionChannel), UserID: event.Member.User.ID}
	switch sub.Name {
	case subCreate:
		item.Action = actionCreate
	case subEdit:
		item.Action = actionEdit
	case subPreview:
		item.Action, item.Key = actionPreview, noValue
	case subShow, subArchive, subList:
		if discord.DeferEphemeral(ctx, session, event.Interaction) == nil {
			c.simple(ctx, session, event.Interaction, sub.Name, item)
		}
		return
	default:
		return
	}
	c.collect(ctx, session, event, data, sub, item)
}

func (c *command) collect(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData, sub *discordgo.ApplicationCommandInteractionDataOption, item pending) {
	file := attachmentOption(data, sub.Options)
	if file == nil {
		c.openModal(ctx, session, event.Interaction, item)
		return
	}
	if discord.DeferEphemeral(ctx, session, event.Interaction) != nil {
		return
	}
	raw, err := c.readAttachment(ctx, file)
	if err != nil {
		c.edit(ctx, session, event.Interaction, "❌ No se pudo leer el archivo: "+err.Error(), nil)
		return
	}
	c.present(ctx, session, event.Interaction, item, raw)
}

func (c *command) autocomplete(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	data := event.ApplicationCommandData()
	if len(data.Options) == 0 {
		return
	}
	typed := ""
	for _, option := range data.Options[0].Options {
		if option.Focused && option.Name == optionKey {
			typed, _ = option.Value.(string)
		}
	}
	page, err := c.manager.List(ctx, messages.ListQuery{Limit: 100})
	if err != nil {
		c.log.Error("autocomplete managed messages", zap.Error(err))
		return
	}
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, autocompleteLimit)
	for _, record := range page.Items {
		if record.State != messages.StateArchived && strings.Contains(record.Key, strings.ToLower(typed)) && len(choices) < autocompleteLimit {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: record.Key, Value: record.Key})
		}
	}
	if err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult, Data: &discordgo.InteractionResponseData{Choices: choices},
	}, discordgo.WithContext(ctx)); err != nil {
		c.log.Error("respond to autocomplete", zap.Error(err))
	}
}

func (c *command) edit(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, content string, components []discordgo.MessageComponent, files ...*discordgo.File) {
	if err := discord.EditResponse(ctx, session, interaction, content, components, files...); err != nil {
		c.log.Error("respond to message command", zap.Error(err))
	}
}

func stringOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if value, ok := option.Value.(string); ok && option.Name == name {
			return value
		}
	}
	return ""
}

func attachmentOption(data discordgo.ApplicationCommandInteractionData, options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.MessageAttachment {
	id := stringOption(options, optionFile)
	if id == "" || data.Resolved == nil {
		return nil
	}
	return data.Resolved.Attachments[id]
}

func (c *command) readAttachment(ctx context.Context, file *discordgo.MessageAttachment) ([]byte, error) {
	if file.Size > maxPayloadBytes {
		return nil, fmt.Errorf("el archivo supera %d KiB", maxPayloadBytes>>10)
	}
	body, err := c.fetch(ctx, file.URL)
	if err == nil && len(body) > maxPayloadBytes {
		err = fmt.Errorf("el archivo supera %d KiB", maxPayloadBytes>>10)
	}
	return body, err
}
