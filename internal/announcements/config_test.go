package announcements

import (
	"testing"
	"time"
)

func TestLoadConfigWithChannel(t *testing.T) {
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "123456789")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.ChannelID != "123456789" || config.Cooldown != 30*time.Minute {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigDisabledWithoutChannel(t *testing.T) {
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.ChannelID != "" {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigRejectsInvalidChannel(t *testing.T) {
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "not-a-snowflake")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() expected error for invalid channel")
	}
}
