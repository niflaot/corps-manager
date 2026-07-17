package app

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DISCORD_BOT_ENVIRONMENT", "development")
	t.Setenv("DISCORD_BOT_HOST", "127.0.0.1")
	t.Setenv("DISCORD_BOT_PORT", "3100")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.Environment != EnvironmentDevelopment || config.Address() != "127.0.0.1:3100" {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigRejectsEnvironment(t *testing.T) {
	t.Setenv("DISCORD_BOT_ENVIRONMENT", "staging")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}
