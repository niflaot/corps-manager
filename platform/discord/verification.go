package discord

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pixelados-net/discord-bot/internal/localization"
	"github.com/pixelados-net/discord-bot/internal/verification"
	"go.uber.org/zap"
)

const (
	memberLifecycleTimeout = 15 * time.Second
	memberJoinOperation    = "join"
	memberRemoveOperation  = "remove"
)

// VerificationGateway applies verification roles and DMs in one guild.
type VerificationGateway struct {
	client  *Client
	catalog *localization.Catalog
}

// NewVerificationGateway creates the guild-scoped verification adapter.
func NewVerificationGateway(client *Client, catalog *localization.Catalog) *VerificationGateway {
	return &VerificationGateway{client: client, catalog: catalog}
}

// MemberState returns the user's current guild membership and assigned roles.
func (gateway *VerificationGateway) MemberState(ctx context.Context, userID string) (verification.MemberState, error) {
	member, err := gateway.client.session.GuildMember(gateway.client.guildID, userID, discordgo.WithContext(ctx))
	if isDiscordErrorCode(err, discordgo.ErrCodeUnknownMember) {
		return verification.MemberState{Present: false}, nil
	}
	if err != nil {
		return verification.MemberState{}, err
	}
	roles := make(map[string]struct{}, len(member.Roles))
	for _, roleID := range member.Roles {
		roles[roleID] = struct{}{}
	}
	return verification.MemberState{Present: true, JoinedAt: member.JoinedAt, RoleIDs: roles}, nil
}

// ValidateRole verifies the target is a non-managed role below the bot.
func (gateway *VerificationGateway) ValidateRole(ctx context.Context, roleID string) error {
	roles, err := gateway.client.session.GuildRoles(gateway.client.guildID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	userID, err := gateway.client.BotUserID(ctx)
	if err != nil {
		return err
	}
	member, err := gateway.client.session.GuildMember(gateway.client.guildID, userID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	owned := make(map[string]bool, len(member.Roles))
	for _, id := range member.Roles {
		owned[id] = true
	}
	botPosition, targetPosition, found := -1, -1, false
	for _, role := range roles {
		if owned[role.ID] && role.Position > botPosition {
			botPosition = role.Position
		}
		if role.ID == roleID {
			if role.Managed || role.ID == gateway.client.guildID {
				return fmt.Errorf("role is managed or @everyone")
			}
			targetPosition, found = role.Position, true
		}
	}
	if !found {
		return fmt.Errorf("role does not exist in configured guild")
	}
	if targetPosition >= botPosition {
		return fmt.Errorf("role must be below the bot's highest role")
	}
	return nil
}

// AddRole assigns one verification role.
func (gateway *VerificationGateway) AddRole(ctx context.Context, userID, roleID string) error {
	return gateway.client.session.GuildMemberRoleAdd(gateway.client.guildID, userID, roleID, discordgo.WithContext(ctx))
}

// RemoveRole removes one verification role.
func (gateway *VerificationGateway) RemoveRole(ctx context.Context, userID, roleID string) error {
	err := gateway.client.session.GuildMemberRoleRemove(gateway.client.guildID, userID, roleID, discordgo.WithContext(ctx))
	if isDiscordErrorCode(err, discordgo.ErrCodeUnknownMember) || isDiscordErrorCode(err, discordgo.ErrCodeUnknownRole) {
		return nil
	}
	return err
}

// SendVerifiedDM sends a Components V2 success message and unverify button.
func (gateway *VerificationGateway) SendVerifiedDM(ctx context.Context, userID string, group verification.Group) error {
	channel, err := gateway.client.session.UserChannelCreate(userID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	components := []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: gateway.catalog.Text(localization.VerificationSuccessKey, "group", group.ButtonLabel)},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{
			Label: gateway.catalog.Text(localization.VerificationUnverifyKey), Style: discordgo.DangerButton,
			CustomID: verification.LeaveCustomIDPrefix + group.ID,
		}}},
	}
	_, err = gateway.client.session.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{Components: components,
		Flags: discordgo.MessageFlagsIsComponentsV2, AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}}, discordgo.WithContext(ctx))
	return err
}

// RegisterVerificationHandlers registers scoped component interaction handling.
func RegisterVerificationHandlers(client *Client, service *verification.Service, catalog *localization.Catalog) func() {
	return client.AddHandler(func(session *discordgo.Session, event *discordgo.InteractionCreate) {
		if event.Type != discordgo.InteractionMessageComponent {
			return
		}
		customID := event.MessageComponentData().CustomID
		groupID, joining := trimInteraction(customID, verification.JoinCustomIDPrefix)
		if !joining {
			var leaving bool
			groupID, leaving = trimInteraction(customID, verification.LeaveCustomIDPrefix)
			if !leaving {
				return
			}
		}
		userID := interactionUserID(event)
		processing := []discordgo.MessageComponent{discordgo.TextDisplay{Content: catalog.Text(localization.VerificationProcessingKey)}}
		if err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Components: processing, Flags: discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2}}); err != nil {
			return
		}
		operationContext, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		err := service.Unverify(operationContext, userID, groupID)
		messageKey := localization.VerificationRemovedKey
		if joining {
			err = service.Verify(operationContext, event.GuildID, userID, groupID)
			messageKey = localization.VerificationSuccessShortKey
		}
		if err != nil {
			messageKey = localization.VerificationFailedKey
		}
		components := []discordgo.MessageComponent{discordgo.TextDisplay{Content: catalog.Text(messageKey)}}
		_, _ = session.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{Components: &components})
	})
}

// RegisterMemberLifecycleHandlers reconciles verification state on guild joins and departures.
func RegisterMemberLifecycleHandlers(client *Client, service *verification.Service) []func() {
	reconcile := func(guildID, userID, operation string) {
		if guildID != client.guildID || userID == "" {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), memberLifecycleTimeout)
		defer cancel()
		if err := service.ReconcileMember(ctx, guildID, userID); err != nil {
			client.log.Error("verification member lifecycle reconcile failed",
				zap.String("operation", operation), zap.String("user", userID), zap.Error(err))
		}
	}
	joined := client.AddHandler(func(_ *discordgo.Session, event *discordgo.GuildMemberAdd) {
		if event.Member != nil && event.User != nil {
			reconcile(event.GuildID, event.User.ID, memberJoinOperation)
		}
	})
	removed := client.AddHandler(func(_ *discordgo.Session, event *discordgo.GuildMemberRemove) {
		if event.Member != nil && event.User != nil {
			reconcile(event.GuildID, event.User.ID, memberRemoveOperation)
		}
	})
	return []func(){joined, removed}
}

func isDiscordErrorCode(err error, code int) bool {
	var restError *discordgo.RESTError
	return errors.As(err, &restError) && restError.Message != nil && restError.Message.Code == code
}

func trimInteraction(value, prefix string) (string, bool) {
	if len(value) <= len(prefix) || value[:len(prefix)] != prefix {
		return "", false
	}
	return value[len(prefix):], true
}

func interactionUserID(event *discordgo.InteractionCreate) string {
	if event.Member != nil && event.Member.User != nil {
		return event.Member.User.ID
	}
	if event.User != nil {
		return event.User.ID
	}
	return ""
}
