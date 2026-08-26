package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/announcements"
	"github.com/niflaot/corps-manager/internal/inactivity"
	"go.uber.org/zap"
)

const (
	inactivityListPageSize = 20
	inactivityListPrefix   = "inactivity:list:page:"
	openingTitle           = "Benny's Motor está abierto al público"
	openingDescription     = "¡Visítanos en la propiedad **1995**!\nUsa `/prop 1995` para llegar."
	openingColor           = 0x2ecc71
)

func inactivityListPage(customID string) (int, bool, bool) {
	if customID == inactivity.ButtonListCustomID {
		return 0, false, true
	}
	if !strings.HasPrefix(customID, inactivityListPrefix) {
		return 0, false, false
	}
	page, err := strconv.Atoi(strings.TrimPrefix(customID, inactivityListPrefix))
	return max(page, 0), true, err == nil
}

func (handler *inactivityInteractionHandler) showInactivityList(session *discordgo.Session,
	event *discordgo.InteractionCreate, page int, update bool) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	entries, err := handler.service.List(ctx)
	if err != nil {
		handler.log.Error("list inactivity registry", zap.Error(err))
		handler.respondList(ctx, session, event.Interaction, "❌ No fue posible consultar el registro.", nil, update)
		return
	}
	pageCount := max((len(entries)+inactivityListPageSize-1)/inactivityListPageSize, 1)
	page = min(page, pageCount-1)
	start := page * inactivityListPageSize
	end := min(start+inactivityListPageSize, len(entries))
	content := renderInactivityList(entries[start:end], len(entries), page, pageCount)
	buttons := inactivityListButtons(page, pageCount)
	handler.respondList(ctx, session, event.Interaction, content, buttons, update)
}

func renderInactivityList(entries []inactivity.Entry, total int, page int, pageCount int) string {
	var content strings.Builder
	fmt.Fprintf(&content, "**Expulsados por inactividad** · Página %d/%d · Total: %d\n", page+1, pageCount, total)
	if len(entries) == 0 {
		content.WriteString("No hay empleados registrados.")
		return content.String()
	}
	content.WriteString("```text\n")
	for _, entry := range entries {
		content.WriteString(entry.Name)
		content.WriteByte('\n')
	}
	content.WriteString("```")
	return content.String()
}

func inactivityListButtons(page int, pageCount int) []discordgo.MessageComponent {
	if pageCount <= 1 {
		return nil
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "Anterior", Style: discordgo.SecondaryButton,
			CustomID: inactivityListPrefix + strconv.Itoa(max(page-1, 0)), Disabled: page == 0},
		discordgo.Button{Label: "Siguiente", Style: discordgo.SecondaryButton,
			CustomID: inactivityListPrefix + strconv.Itoa(min(page+1, pageCount-1)), Disabled: page+1 >= pageCount},
	}}}
}

func (handler *inactivityInteractionHandler) respondList(ctx context.Context, session *discordgo.Session,
	interaction *discordgo.Interaction, content string, components []discordgo.MessageComponent, update bool) {
	responseType := discordgo.InteractionResponseChannelMessageWithSource
	flags := discordgo.MessageFlagsEphemeral
	if update {
		responseType = discordgo.InteractionResponseUpdateMessage
		flags = 0
	}
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: responseType,
		Data: &discordgo.InteractionResponseData{Content: content, Components: components, Flags: flags,
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}},
		discordgo.WithContext(ctx))
	if err != nil {
		handler.log.Error("respond with inactivity registry list", zap.Error(err))
	}
}

func (handler *inactivityInteractionHandler) announceOpening(session *discordgo.Session,
	event *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	actor := interactionActor(event)
	state, err := handler.announcementService.AnnounceOpening(ctx, actor)
	if err != nil {
		var active *announcements.CooldownActiveError
		if errors.As(err, &active) {
			message := fmt.Sprintf("⏳ La apertura ya fue anunciada. Podrás repetirla <t:%d:R>.",
				active.State.AvailableAt.Unix())
			handler.respondWithContext(ctx, session, event.Interaction, message)
			return
		}
		if errors.Is(err, announcements.ErrDisabled) {
			handler.respondWithContext(ctx, session, event.Interaction,
				"❌ El canal de anuncios no está configurado.")
			return
		}
		handler.log.Error("publish business opening announcement", zap.String("actor", actor), zap.Error(err))
		handler.respondWithContext(ctx, session, event.Interaction, "❌ No fue posible publicar la apertura.")
		return
	}
	message := fmt.Sprintf("✅ La apertura fue anunciada. Disponible nuevamente <t:%d:R>.", state.AvailableAt.Unix())
	handler.respondWithContext(ctx, session, event.Interaction, message)
}

// OpeningGateway publishes business-opening announcements through Discord.
type OpeningGateway struct{ client *Client }

// NewOpeningGateway creates a Discord opening-announcement gateway.
func NewOpeningGateway(client *Client) *OpeningGateway { return &OpeningGateway{client: client} }

// SendOpening publishes one attributed opening announcement in the configured channel.
func (gateway *OpeningGateway) SendOpening(ctx context.Context, channelID string, actor string) error {
	if _, err := gateway.client.session.ChannelMessageSendComplex(channelID, openingAnnouncement(actor),
		discordgo.WithContext(ctx)); err != nil {
		return fmt.Errorf("send Discord opening announcement: %w", err)
	}
	return nil
}

func openingAnnouncement(actor string) *discordgo.MessageSend {
	return &discordgo.MessageSend{Content: "@everyone", Embeds: []*discordgo.MessageEmbed{{
		Title: openingTitle, Description: openingDescription, Color: openingColor,
		Footer: &discordgo.MessageEmbedFooter{Text: "Anunciado por: " + actor},
	}}, AllowedMentions: &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeEveryone},
	}}
}

func interactionActor(event *discordgo.InteractionCreate) string {
	if event.Member != nil && strings.TrimSpace(event.Member.Nick) != "" {
		return strings.TrimSpace(event.Member.Nick)
	}
	user := event.User
	if event.Member != nil && event.Member.User != nil {
		user = event.Member.User
	}
	if user != nil && strings.TrimSpace(user.GlobalName) != "" {
		return strings.TrimSpace(user.GlobalName)
	}
	if user != nil && strings.TrimSpace(user.Username) != "" {
		return strings.TrimSpace(user.Username)
	}
	return "Usuario desconocido"
}
