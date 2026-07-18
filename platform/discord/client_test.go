package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func TestLoadConfigRequiresToken(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	t.Setenv("DISCORD_BOT_GUILD_ID", "123")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

func TestLoadConfigRequiresGuild(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "test-token")
	t.Setenv("DISCORD_BOT_GUILD_ID", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

func TestLoadConfigAndNew(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "test-token")
	t.Setenv("DISCORD_BOT_GUILD_ID", "123")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	client, err := New(config, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.SDK() == nil || client.Connected() {
		t.Fatalf("unexpected client state: %#v", client)
	}
	if config.Intents&discordgo.IntentsGuildMembers == 0 {
		t.Fatal("GUILD_MEMBERS intent is disabled")
	}
}
