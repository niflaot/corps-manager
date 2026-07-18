// Package discord contains the Discord gateway adapter.
package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
)

// Config contains Discord bot connection settings.
type Config struct {
	// Token authenticates the Discord bot.
	Token string `env:"DISCORD_BOT_TOKEN"`
	// GuildID is the only Discord guild this process may manage.
	GuildID string `env:"DISCORD_BOT_GUILD_ID"`
	// Intents selects the Discord gateway events consumed by the bot.
	Intents discordgo.Intent `env:"-"`
}

// LoadConfig reads and validates the mandatory Discord bot configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.Token = strings.TrimSpace(config.Token)
	config.GuildID = strings.TrimSpace(config.GuildID)
	if config.Token == "" {
		return Config{}, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}
	if !snowflakePattern.MatchString(config.GuildID) {
		return Config{}, fmt.Errorf("DISCORD_BOT_GUILD_ID must be a Discord snowflake")
	}
	config.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages
	return config, nil
}
