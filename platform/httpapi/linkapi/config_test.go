package linkapi

import (
	"testing"

	"github.com/pixelados-net/discord-bot/platform/discordoauth"
)

func TestLoadConfigRequiresAllowlistedCompletionURLs(t *testing.T) {
	oauthConfig := discordoauth.Config{Enabled: true, PublicURL: "https://discord-api.example.test"}
	t.Setenv("DISCORD_BOT_OAUTH_COMPLETION_URLS", `{"pixelados-links":"https://pixelados.example.test/account/links/result"}`)
	config, err := LoadConfig(oauthConfig)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	destination, err := config.completionURL("pixelados-links", "secret result")
	if err != nil || destination != "https://pixelados.example.test/account/links/result?code=secret+result" {
		t.Fatalf("completionURL() = %q, error = %v", destination, err)
	}
	t.Setenv("DISCORD_BOT_OAUTH_COMPLETION_URLS", `{"pixelados-links":"https://pixelados.example.test/result#fragment"}`)
	if _, err = LoadConfig(oauthConfig); err == nil {
		t.Fatal("fragment completion LoadConfig() error = nil")
	}
}
