package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoadConfigRequiresToken(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	t.Setenv("DISCORD_BOT_GUILD_ID", "123")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

func TestNewRoutesDiscordGoLogsThroughZap(t *testing.T) {
	previousLogger := discordgo.Logger
	t.Cleanup(func() { discordgo.Logger = previousLogger })
	core, observed := observer.New(zap.DebugLevel)
	client, err := New(Config{Token: "test-token", GuildID: "123"}, zap.New(core))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	discordgo.Logger(discordgo.LogWarning, 0, "rate limited for %d seconds", 2)
	entries := observed.All()
	if client.SDK().LogLevel != discordgo.LogDebug || len(entries) != 1 {
		t.Fatalf("DiscordGo logging state = %d, entries = %#v", client.SDK().LogLevel, entries)
	}
	if entries[0].Level != zap.WarnLevel || entries[0].LoggerName != discordLoggerName || entries[0].Message != "rate limited for 2 seconds" {
		t.Fatalf("DiscordGo log entry = %#v", entries[0])
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
	if config.Intents&discordgo.IntentsGuildMembers != 0 {
		t.Fatal("GUILD_MEMBERS intent is unexpectedly enabled")
	}
}
