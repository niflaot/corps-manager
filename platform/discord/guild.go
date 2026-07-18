package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// GuildGateway reads public-facing guild statistics.
type GuildGateway struct {
	client *Client
}

// NewGuildGateway creates a guild statistics gateway.
func NewGuildGateway(client *Client) *GuildGateway { return &GuildGateway{client: client} }

// MemberCount returns the configured guild's approximate member and online presence counts.
func (gateway *GuildGateway) MemberCount(ctx context.Context) (int, int, error) {
	guild, err := gateway.client.session.GuildWithCounts(gateway.client.guildID, discordgo.WithContext(ctx))
	if err != nil {
		return 0, 0, fmt.Errorf("read guild statistics: %w", err)
	}
	return guild.ApproximateMemberCount, guild.ApproximatePresenceCount, nil
}
