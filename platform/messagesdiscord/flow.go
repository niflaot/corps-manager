package messagesdiscord

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/zap"
)

const (
	payloadInputID = "payload"
	// modalInputLimit is Discord's maximum text input length.
	modalInputLimit  = 4000
	modalTitle       = "Mensaje gestionado"
	modalPlaceholder = `{"components":[{"type":10,"content":"Hola"}]}`
)

func (c *command) openModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, item pending) {
	prefill := ""
	if item.Action == actionEdit {
		record, err := c.manager.Get(ctx, item.Key)
		if err != nil || record.State == messages.StateArchived {
			_ = discord.RespondEphemeral(ctx, session, interaction, c.describe(err, "El mensaje no existe o está archivado."))
			return
		}
		encoded, _ := json.Marshal(record.Payload)
		if utf8.RuneCount(encoded) > modalInputLimit {
			_ = discord.RespondEphemeral(ctx, session, interaction, "El JSON actual supera 4000 caracteres; expórtalo con `/mensaje ver` y adjúntalo en `archivo`.")
			return
		}
		prefill = string(encoded)
	}
	channel := item.ChannelID
	if channel == "" {
		channel = noValue
	}
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{
		CustomID: strings.Join([]string{"msg", "m", string(item.Action), item.Key, channel}, ":"), Title: modalTitle,
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{
			CustomID: payloadInputID, Label: "JSON del mensaje", Style: discordgo.TextInputParagraph, Required: true,
			MaxLength: modalInputLimit, Value: prefill, Placeholder: modalPlaceholder}}}},
	}}, discordgo.WithContext(ctx))
	if err != nil {
		c.log.Error("open message modal", zap.Error(err))
	}
}

func (c *command) modal(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	data := event.ModalSubmitData()
	parts := strings.Split(data.CustomID, ":")
	if len(parts) != 5 || !strings.HasPrefix(data.CustomID, modalPrefix) {
		return
	}
	item := pending{Action: action(parts[2]), Key: parts[3], UserID: event.Member.User.ID}
	if parts[4] != noValue {
		item.ChannelID = parts[4]
	}
	if discord.DeferEphemeral(ctx, session, event.Interaction) == nil {
		c.present(ctx, session, event.Interaction, item, []byte(modalText(data)))
	}
}

func modalText(data discordgo.ModalSubmitInteractionData) string {
	for _, component := range data.Components {
		if row, ok := component.(*discordgo.ActionsRow); ok {
			for _, child := range row.Components {
				if input, ok := child.(*discordgo.TextInput); ok && input.CustomID == payloadInputID {
					return input.Value
				}
			}
		}
	}
	return ""
}

func (c *command) present(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, item pending, raw []byte) {
	payload, err := parsePayload(raw)
	if err != nil {
		c.edit(ctx, session, interaction, "❌ "+truncate(err.Error()), nil)
		return
	}
	item.Payload = payload
	if problem := c.check(ctx, &item); problem != "" {
		c.edit(ctx, session, interaction, "❌ "+problem, nil)
		return
	}
	components, err := previewComponents(payload)
	if err == nil {
		_, err = session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{Components: components,
			Flags:           discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}, discordgo.WithContext(ctx))
	}
	if err != nil {
		c.log.Warn("discord rejected message preview", zap.Error(err))
		c.edit(ctx, session, interaction, "❌ Discord rechazó la vista previa: "+truncate(err.Error()), nil)
		return
	}
	summary := fmt.Sprintf("✅ JSON válido · %d componentes de primer nivel. La vista previa está debajo; sus botones aparecen desactivados.", len(payload.Components))
	if item.Action == actionPreview {
		c.edit(ctx, session, interaction, summary, nil)
		return
	}
	id, err := c.pending.put(item)
	if err != nil {
		c.edit(ctx, session, interaction, "Hay demasiadas vistas previas pendientes; inténtalo en unos minutos.", nil)
		return
	}
	verb := map[action]string{actionCreate: "Crear", actionEdit: "Reemplazar"}[item.Action]
	c.edit(ctx, session, interaction, fmt.Sprintf("%s\n**%s** `%s` en <#%s>. ¿Confirmas?", summary, verb, item.Key, item.ChannelID), confirmRow(id))
}

func (c *command) check(ctx context.Context, item *pending) string {
	if item.Action == actionPreview {
		return ""
	}
	record, err := c.manager.Get(ctx, item.Key)
	exists := err == nil && record.State != messages.StateArchived
	switch {
	case item.Action == actionCreate && exists:
		return "Ya existe un mensaje con esa clave; usa `/mensaje editar`."
	case item.Action == actionEdit && !exists:
		return c.describe(err, "El mensaje no existe o está archivado.")
	case item.Action == actionEdit && item.ChannelID == "":
		item.ChannelID = record.ChannelID
	}
	definition := messages.Definition{Key: item.Key, GuildID: c.guildID, ChannelID: item.ChannelID, Payload: item.Payload}
	if err := definition.Validate(); err != nil {
		return truncate(err.Error())
	}
	return ""
}

func (c *command) component(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	customID := event.MessageComponentData().CustomID
	confirm, cancel := strings.HasPrefix(customID, confirmPrefix), strings.HasPrefix(customID, cancelPrefix)
	if !confirm && !cancel || discord.DeferUpdate(ctx, session, event.Interaction) != nil {
		return
	}
	userID := event.Member.User.ID
	if cancel {
		c.pending.discard(strings.TrimPrefix(customID, cancelPrefix), userID)
		c.edit(ctx, session, event.Interaction, "Operación cancelada.", nil)
		return
	}
	item, result := c.pending.take(strings.TrimPrefix(customID, confirmPrefix), userID)
	switch result {
	case takeMissing:
		c.edit(ctx, session, event.Interaction, "La vista previa expiró; ejecuta el comando de nuevo.", nil)
	case takeForeign:
		c.log.Warn("foreign message confirmation", zap.String("user_id", userID))
	default:
		c.edit(ctx, session, event.Interaction, c.execute(ctx, item, event.ID), nil)
	}
}

func confirmRow(id string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "Confirmar", Style: discordgo.SuccessButton, CustomID: confirmPrefix + id},
		discordgo.Button{Label: "Cancelar", Style: discordgo.SecondaryButton, CustomID: cancelPrefix + id},
	}}}
}

func truncate(text string) string {
	if utf8.RuneCountInString(text) <= 1500 {
		return text
	}
	return string([]rune(text)[:1500]) + "…"
}
