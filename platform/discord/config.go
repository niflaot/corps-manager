// Package discord contains the Discord gateway adapter.
package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v11"
)

// Config contains Discord bot connection settings.
type Config struct {
	// Token authenticates the Discord bot. An empty token disables the gateway.
	Token string `env:"DISCORD_BOT_TOKEN" envDefault:""`
	// Intents selects the Discord gateway events consumed by the bot.
	Intents discordgo.Intent `env:"-"`
}

// LoadConfig reads Discord configuration from DISCORD_BOT_TOKEN.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	config.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages
	return config, err
}
