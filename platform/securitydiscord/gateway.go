// Package securitydiscord adapts guild security to Discord.
package securitydiscord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/platform/discord"
)

const banReason = "anti-bot: posted in no-escribir"
const historyPageSize = 100
const historyPageLimit = 10

// Gateway implements guild-scoped security actions.
type Gateway struct{ client *discord.Client }

// NewGateway creates the security adapter without opening another session.
func NewGateway(client *discord.Client) *Gateway { return &Gateway{client: client} }

// GrantRole validates that the role is assignable and grants it idempotently.
func (gateway *Gateway) GrantRole(ctx context.Context, userID, roleID string) error {
	roles, err := gateway.client.SDK().GuildRoles(gateway.client.GuildID(), discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.ID != roleID {
			continue
		}
		if role.Managed || role.ID == gateway.client.GuildID() {
			return fmt.Errorf("verification role cannot be assigned")
		}
		return gateway.client.SDK().GuildMemberRoleAdd(gateway.client.GuildID(), userID, roleID, discordgo.WithContext(ctx))
	}
	return fmt.Errorf("verification role does not exist in the configured guild")
}

// Ban bans an author without deleting historical messages.
func (gateway *Gateway) Ban(ctx context.Context, userID string) error {
	return gateway.client.SDK().GuildBanCreateWithReason(gateway.client.GuildID(), userID, banReason, 0, discordgo.WithContext(ctx))
}

// PruneWarnings keeps the canonical warning and removes other messages authored by this bot.
func (gateway *Gateway) PruneWarnings(ctx context.Context, channelID, messageID string) error {
	botID, err := gateway.client.BotUserID(ctx)
	if err != nil {
		return err
	}
	// Never delete an old warning until its replacement is confirmed to exist.
	canonical, err := gateway.client.SDK().ChannelMessage(channelID, messageID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	if canonical.Author == nil || canonical.Author.ID != botID {
		return fmt.Errorf("anti-bot warning is not owned by this bot")
	}
	before := ""
	for range historyPageLimit {
		items, err := gateway.client.SDK().ChannelMessages(channelID, historyPageSize, before, "", "", discordgo.WithContext(ctx))
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.ID != messageID && item.Author != nil && item.Author.ID == botID {
				if err := gateway.client.SDK().ChannelMessageDelete(channelID, item.ID, discordgo.WithContext(ctx)); err != nil {
					return err
				}
			}
		}
		if len(items) < historyPageSize {
			return nil
		}
		before = items[len(items)-1].ID
	}
	return nil
}
