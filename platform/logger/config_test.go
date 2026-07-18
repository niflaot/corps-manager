package logger

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("DISCORD_BOT_LOG_LEVEL", "debug")
	t.Setenv("DISCORD_BOT_LOG_FORMAT", "json")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.Level != zapcore.DebugLevel || config.Format != FormatJSON {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigRejectsFormat(t *testing.T) {
	t.Setenv("DISCORD_BOT_LOG_FORMAT", "xml")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

func TestLoadConfigRejectsLevel(t *testing.T) {
	t.Setenv("DISCORD_BOT_LOG_LEVEL", "verbose")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}
