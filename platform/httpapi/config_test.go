package httpapi

import "testing"

func TestLoadConfigRequiresPrivateAPIKey(t *testing.T) {
	t.Setenv("DISCORD_BOT_API_KEY", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("DISCORD_BOT_API_KEY", "key")
	t.Setenv("DISCORD_BOT_HTTP_BODY_LIMIT", "2048")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.APIKey != "key" || config.BodyLimit != 2048 {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}
