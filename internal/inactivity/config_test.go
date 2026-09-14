package inactivity

import "testing"

func TestLoadConfigWithAnnouncementChannels(t *testing.T) {
	t.Setenv("DISCORD_BOT_INACTIVITY_ENABLED", "true")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CHANNEL_ID", "111")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "222")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CONTROL_CHANNEL_ID", "333")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.ChannelID != "111" || config.AnnouncementChannelID != "222" || config.AnnouncementControlChannelID != "333" {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigWithoutAnnouncementChannel(t *testing.T) {
	t.Setenv("DISCORD_BOT_INACTIVITY_ENABLED", "true")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CHANNEL_ID", "111")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CONTROL_CHANNEL_ID", "")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.AnnouncementChannelID != "" || config.AnnouncementControlChannelID != "" {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigRequiresControlChannelWhenAnnouncementChannelSet(t *testing.T) {
	t.Setenv("DISCORD_BOT_INACTIVITY_ENABLED", "true")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CHANNEL_ID", "111")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "222")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CONTROL_CHANNEL_ID", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() expected error for missing control channel")
	}
}

func TestLoadConfigDisabledDoesNotRequireChannels(t *testing.T) {
	t.Setenv("DISCORD_BOT_INACTIVITY_ENABLED", "false")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CHANNEL_ID", "")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CHANNEL_ID", "")
	t.Setenv("DISCORD_BOT_ANNOUNCEMENT_CONTROL_CHANNEL_ID", "")
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
}
