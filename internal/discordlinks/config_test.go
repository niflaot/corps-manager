package discordlinks

import "testing"

func TestLoadConfigValidatesDurations(t *testing.T) {
	t.Setenv("DISCORD_BOT_OAUTH_INTENT_TTL", "0s")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
	t.Setenv("DISCORD_BOT_OAUTH_INTENT_TTL", "1m")
	t.Setenv("DISCORD_BOT_OAUTH_RESULT_TTL", "30s")
	config, err := LoadConfig()
	if err != nil || config.IntentTTL.String() != "1m0s" || config.ResultTTL.String() != "30s" {
		t.Fatalf("LoadConfig() = %#v, error = %v", config, err)
	}
}
