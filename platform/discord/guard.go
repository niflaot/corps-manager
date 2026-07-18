package discord

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
)

const trapBanReason = "verification guard anti-bot trap"

// GuardGateway reconciles verification and anti-bot channels.
type GuardGateway struct {
	client      *Client
	trapChannel atomic.Value
}

// NewGuardGateway creates a guild-scoped guard adapter.
func NewGuardGateway(client *Client) *GuardGateway { return &GuardGateway{client: client} }

// MakeReadOnly keeps the verification channel visible and non-writable to members.
func (gateway *GuardGateway) MakeReadOnly(ctx context.Context, channelID string) error {
	channel, err := gateway.client.session.Channel(channelID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	if channel.GuildID != gateway.client.guildID {
		return errors.New("verification channel is outside configured guild")
	}
	allow := int64(discordgo.PermissionViewChannel | discordgo.PermissionReadMessageHistory)
	deny := int64(discordgo.PermissionSendMessages | discordgo.PermissionAddReactions | discordgo.PermissionCreatePublicThreads |
		discordgo.PermissionCreatePrivateThreads | discordgo.PermissionSendMessagesInThreads |
		discordgo.PermissionUseApplicationCommands | discordgo.PermissionSendVoiceMessages | discordgo.PermissionSendPolls)
	everyoneFound := false
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.ID == gateway.client.guildID {
			everyoneFound = true
			overwrite.Allow |= allow
		}
		overwrite.Allow &^= deny
		overwrite.Deny |= deny
		if err := gateway.client.session.ChannelPermissionSet(channelID, overwrite.ID, overwrite.Type, overwrite.Allow, overwrite.Deny, discordgo.WithContext(ctx)); err != nil {
			return err
		}
	}
	if !everyoneFound {
		return gateway.client.session.ChannelPermissionSet(channelID, gateway.client.guildID, discordgo.PermissionOverwriteTypeRole, allow, deny, discordgo.WithContext(ctx))
	}
	return nil
}

// ReconcileTrap ensures the visible writable last channel and its Components V2 warning.
func (gateway *GuardGateway) ReconcileTrap(ctx context.Context, name, warning, channelID, messageID string) (string, string, error) {
	channels, err := gateway.client.session.GuildChannels(gateway.client.guildID, discordgo.WithContext(ctx))
	if err != nil {
		return "", "", err
	}
	var channel *discordgo.Channel
	maximumPosition := 0
	for _, candidate := range channels {
		if candidate.Position > maximumPosition {
			maximumPosition = candidate.Position
		}
		if candidate.Type == discordgo.ChannelTypeGuildText && (candidate.ID == channelID || channel == nil && candidate.Name == name) {
			channel = candidate
		}
	}
	allow := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory)
	if channel == nil {
		channel, err = gateway.client.session.GuildChannelCreateComplex(gateway.client.guildID, discordgo.GuildChannelCreateData{
			Name: name, Type: discordgo.ChannelTypeGuildText, Position: maximumPosition + 1,
			PermissionOverwrites: []*discordgo.PermissionOverwrite{{ID: gateway.client.guildID, Type: discordgo.PermissionOverwriteTypeRole, Allow: allow}},
		}, discordgo.WithContext(ctx))
		if err != nil {
			return "", "", err
		}
		position := maximumPosition + 1
		if _, err = gateway.client.session.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{Position: &position}, discordgo.WithContext(ctx)); err != nil {
			return "", "", err
		}
	} else {
		position := maximumPosition + 1
		if _, err = gateway.client.session.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{Name: name, Position: &position}, discordgo.WithContext(ctx)); err != nil {
			return "", "", err
		}
		if err = gateway.client.session.ChannelPermissionSet(channel.ID, gateway.client.guildID, discordgo.PermissionOverwriteTypeRole, allow, 0, discordgo.WithContext(ctx)); err != nil {
			return "", "", err
		}
		for _, overwrite := range channel.PermissionOverwrites {
			if overwrite.ID != gateway.client.guildID {
				if err = gateway.client.session.ChannelPermissionDelete(channel.ID, overwrite.ID, discordgo.WithContext(ctx)); err != nil {
					return "", "", err
				}
			}
		}
	}
	message, getErr := gateway.client.session.ChannelMessage(channel.ID, messageID, discordgo.WithContext(ctx))
	components := []discordgo.MessageComponent{discordgo.TextDisplay{Content: warning}}
	botID, botErr := gateway.client.BotUserID(ctx)
	if botErr != nil {
		return "", "", botErr
	}
	if getErr != nil || message.Author == nil || message.Author.ID != botID {
		message, err = gateway.client.session.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{Components: components,
			Flags: discordgo.MessageFlagsIsComponentsV2, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}, discordgo.WithContext(ctx))
	} else if !trapWarningMatches(message, warning) {
		message, err = gateway.client.session.ChannelMessageEditComplex(&discordgo.MessageEdit{ID: message.ID, Channel: channel.ID,
			Components: &components, Flags: discordgo.MessageFlagsIsComponentsV2}, discordgo.WithContext(ctx))
	}
	if err != nil {
		return "", "", err
	}
	gateway.trapChannel.Store(channel.ID)
	return channel.ID, message.ID, nil
}

func trapWarningMatches(message *discordgo.Message, warning string) bool {
	if message.Flags&discordgo.MessageFlagsIsComponentsV2 == 0 || len(message.Components) != 1 {
		return false
	}
	text, ok := message.Components[0].(*discordgo.TextDisplay)
	return ok && text.Content == warning
}

// RegisterTrapHandler bans any non-bot author posting in the reconciled trap channel.
func (gateway *GuardGateway) RegisterTrapHandler() func() {
	return gateway.client.AddHandler(func(session *discordgo.Session, message *discordgo.MessageCreate) {
		stored := gateway.trapChannel.Load()
		if stored == nil || message.GuildID != gateway.client.guildID || message.ChannelID != stored.(string) || message.Author == nil {
			return
		}
		botID, err := gateway.client.BotUserID(context.Background())
		if err != nil || message.Author.ID == botID {
			return
		}
		if err := session.GuildBanCreateWithReason(gateway.client.guildID, message.Author.ID, trapBanReason, 0); err != nil {
			gateway.client.log.Error("verification trap ban failed")
		}
	})
}
