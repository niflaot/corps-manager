package e2e

import (
	"strings"
	"testing"
)

func TestBaseWiringRequiresDiscordTokenE2E(t *testing.T) {
	result := runHarness(t, []string{"serve"}, "DISCORD_BOT_TOKEN=")
	if result.err == nil {
		t.Fatal("serve succeeded without DISCORD_BOT_TOKEN")
	}
	if !strings.Contains(result.output, "DISCORD_BOT_TOKEN is required") {
		t.Fatalf("output = %q", result.output)
	}
}

func TestBaseWiringRequiresDiscordGuildE2E(t *testing.T) {
	result := runHarness(t, []string{"serve"}, "DISCORD_BOT_TOKEN=test-token", "DISCORD_BOT_GUILD_ID=")
	if result.err == nil {
		t.Fatal("serve succeeded without DISCORD_BOT_GUILD_ID")
	}
	if !strings.Contains(result.output, "DISCORD_BOT_GUILD_ID must be a Discord snowflake") {
		t.Fatalf("output = %q", result.output)
	}
}

func TestVersionE2E(t *testing.T) {
	result := runHarness(t, []string{"--version"})
	if result.err != nil {
		t.Fatalf("version error = %v, output = %q", result.err, result.output)
	}
	if result.output != "discord-bot v1.1.0\n" {
		t.Fatalf("output = %q", result.output)
	}
}
