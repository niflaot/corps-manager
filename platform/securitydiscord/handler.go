package securitydiscord

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/security"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/zap"
)

const interactionTimeout = 15 * time.Second

type handler struct {
	ctx     context.Context
	client  *discord.Client
	service *security.Service
	log     *zap.Logger
}

func (handler *handler) interaction(session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionMessageComponent || event.MessageComponentData().CustomID != security.VerifyButtonID {
		return
	}
	if event.GuildID != handler.client.GuildID() || event.Message == nil || event.Member == nil || event.Member.User == nil {
		return
	}
	ctx, cancel := context.WithTimeout(handler.ctx, interactionTimeout)
	defer cancel()
	if err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}, discordgo.WithContext(ctx)); err != nil {
		handler.log.Error("acknowledge verification", zap.Error(err))
		return
	}
	content := "✅ Verificación completada. Ya tienes el rol asignado."
	if err := handler.service.Verify(ctx, event.GuildID, event.ChannelID, event.Message.ID, event.Member.User.ID); err != nil {
		handler.log.Error("verify guild member", zap.String("user_id", event.Member.User.ID), zap.Error(err))
		content = "No fue posible verificarte. El control puede estar desactualizado o faltar permisos; avisa a un administrador."
	}
	if _, err := session.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{Content: &content}, discordgo.WithContext(ctx)); err != nil {
		handler.log.Error("respond to verification", zap.Error(err))
	}
}

func (handler *handler) message(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event.Message == nil || event.Author == nil || !handler.service.IsTrap(event.GuildID, event.ChannelID) {
		return
	}
	ctx, cancel := context.WithTimeout(handler.ctx, interactionTimeout)
	defer cancel()
	botID, err := handler.client.BotUserID(ctx)
	if err == nil {
		err = handler.service.HandleMessage(ctx, event.GuildID, event.ChannelID, event.Author.ID, botID)
	}
	if err != nil {
		handler.log.Error("ban anti-bot channel author", zap.String("user_id", event.Author.ID), zap.Error(err))
	} else if event.Author.ID != botID {
		handler.log.Info("anti-bot member banned", zap.String("user_id", event.Author.ID))
	}
}
