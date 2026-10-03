package securitydiscord

import (
	"context"
	"reflect"
	"sort"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/security"
)

// EnsureTrap restores one writable trap under a category at the bottom of the guild.
func (gateway *Gateway) EnsureTrap(ctx context.Context, preferred string) (string, error) {
	session := gateway.client.SDK()
	guildID := gateway.client.GuildID()
	channels, err := session.GuildChannels(guildID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].ID < channels[j].ID })
	var trap, category *discordgo.Channel
	bottom := 0
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildCategory {
			bottom = max(bottom, channel.Position)
			if channel.Name == security.TrapCategoryName && category == nil {
				category = channel
			}
		}
		if channel.Type == discordgo.ChannelTypeGuildText && channel.ID == preferred {
			trap = channel
		}
	}
	if trap == nil {
		for _, channel := range channels {
			if channel.Type == discordgo.ChannelTypeGuildText && channel.Name == security.TrapChannelName {
				trap = channel
				break
			}
		}
	}
	if category == nil {
		category, err = session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
			Name: security.TrapCategoryName, Type: discordgo.ChannelTypeGuildCategory, Position: bottom + 1,
		}, discordgo.WithContext(ctx))
		if err != nil {
			return "", err
		}
	} else if category.Position < bottom {
		position := bottom + 1
		if _, err := session.ChannelEdit(category.ID, &discordgo.ChannelEdit{Position: &position}, discordgo.WithContext(ctx)); err != nil {
			return "", err
		}
	}
	botID, err := gateway.client.BotUserID(ctx)
	if err != nil {
		return "", err
	}
	permissions := trapPermissions(guildID, botID)
	if trap == nil {
		trap, err = session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
			Name: security.TrapChannelName, Type: discordgo.ChannelTypeGuildText, Topic: security.TrapWarning,
			ParentID: category.ID, PermissionOverwrites: permissions,
		}, discordgo.WithContext(ctx))
		if err != nil {
			return "", err
		}
	}
	position := 0
	for _, channel := range channels {
		if channel.ParentID == category.ID && channel.ID != trap.ID {
			position = max(position, channel.Position+1)
		}
	}
	if trap.Name != security.TrapChannelName || trap.ParentID != category.ID || trap.Topic != security.TrapWarning || trap.NSFW || trap.RateLimitPerUser != 0 || trap.Position < position || !samePermissions(trap.PermissionOverwrites, permissions) {
		disabled, slowmode := false, 0
		if _, err := session.ChannelEdit(trap.ID, &discordgo.ChannelEdit{
			Name: security.TrapChannelName, ParentID: category.ID, Topic: security.TrapWarning, Position: &position,
			PermissionOverwrites: permissions, NSFW: &disabled, RateLimitPerUser: &slowmode,
		}, discordgo.WithContext(ctx)); err != nil {
			return "", err
		}
	}
	for _, channel := range channels {
		if channel.Name == security.TrapChannelName && channel.ID != trap.ID {
			if _, err := session.ChannelDelete(channel.ID, discordgo.WithContext(ctx)); err != nil {
				return "", err
			}
		}
	}
	return trap.ID, nil
}

func trapPermissions(guildID, botID string) []*discordgo.PermissionOverwrite {
	const readable = discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory
	const noThreads = discordgo.PermissionCreatePublicThreads | discordgo.PermissionCreatePrivateThreads | discordgo.PermissionSendMessagesInThreads
	return []*discordgo.PermissionOverwrite{
		{ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Allow: readable, Deny: noThreads},
		{ID: botID, Type: discordgo.PermissionOverwriteTypeMember, Allow: readable | discordgo.PermissionManageMessages | discordgo.PermissionManageChannels},
	}
}

func samePermissions(left, right []*discordgo.PermissionOverwrite) bool {
	copyLeft := append([]*discordgo.PermissionOverwrite(nil), left...)
	copyRight := append([]*discordgo.PermissionOverwrite(nil), right...)
	sort.Slice(copyLeft, func(i, j int) bool { return copyLeft[i].ID < copyLeft[j].ID })
	sort.Slice(copyRight, func(i, j int) bool { return copyRight[i].ID < copyRight[j].ID })
	return reflect.DeepEqual(copyLeft, copyRight)
}
