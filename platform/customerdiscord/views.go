package customerdiscord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (handler *handler) showDetail(session *discordgo.Session, event *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	item, err := handler.service.Get(ctx, input(event.ModalSubmitData(), nameInputID))
	if err != nil {
		handler.respondError(ctx, session, event.Interaction, "consultar el cliente", err)
		return
	}
	var content strings.Builder
	fmt.Fprintf(&content, "**Cliente:** `%s`\n**Visitas:** %d\n**Gasto acumulado:** $%d\n**Personas que lo atendieron:** %d\n",
		item.Name, item.Visits, item.TotalSpent, len(item.Attendants))
	if len(item.Attendants) > 0 {
		content.WriteString("```text\n")
		for _, attendant := range item.Attendants[:min(len(item.Attendants), 15)] {
			fmt.Fprintf(&content, "%-30s %3d · ID %s\n", attendant.DisplayName, attendant.Visits,
				attendant.DiscordUserID)
		}
		content.WriteString("```")
		if len(item.Attendants) > 15 {
			fmt.Fprintf(&content, "Y %d personas más.", len(item.Attendants)-15)
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
		handler.log.Error("respond to customer interaction", zap.Error(err))
	}
}
