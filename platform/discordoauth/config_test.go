package discordoauth

import "testing"

func TestLoadConfigAllowsDisabledOAuth(t *testing.T) {
	t.Setenv("DISCORD_BOT_OAUTH_ENABLED", "false")
	t.Setenv("DISCORD_BOT_OAUTH_CLIENT_ID", "")
	config, err := LoadConfig()
	if err != nil || config.Enabled {
		t.Fatalf("LoadConfig() = %#v, error = %v", config, err)
	}
}

func TestLoadConfigRequiresConfidentialHTTPSClient(t *testing.T) {
	t.Setenv("DISCORD_BOT_OAUTH_ENABLED", "true")
	t.Setenv("DISCORD_BOT_OAUTH_CLIENT_ID", "123")
	t.Setenv("DISCORD_BOT_OAUTH_CLIENT_SECRET", "secret")
	t.Setenv("DISCORD_BOT_OAUTH_PUBLIC_URL", "https://discord-api.example.test")
	config, err := LoadConfig()
	if err != nil || config.CallbackURL() != "https://discord-api.example.test/oauth/discord/callback" {
		t.Fatalf("LoadConfig() = %#v, error = %v", config, err)
	}
	t.Setenv("DISCORD_BOT_OAUTH_PUBLIC_URL", "http://remote.example.test")
	if _, err = LoadConfig(); err == nil {
		t.Fatal("insecure LoadConfig() error = nil")
	}
}
