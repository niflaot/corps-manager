package discord

import (
	"testing"

	"go.uber.org/zap"
)

func TestLoadConfigRequiresToken(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

func TestLoadConfigAndNew(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "test-token")
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
}
