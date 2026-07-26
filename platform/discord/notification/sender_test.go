package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pixelados-net/discord-bot/internal/localization"
	verificationnotification "github.com/pixelados-net/discord-bot/internal/verification/notification"
	discordplatform "github.com/pixelados-net/discord-bot/platform/discord"
	"go.uber.org/zap"
)

// TestSenderUsesLocalizedGroupKeyAndStableNonce verifies translation and delivery idempotency.
func TestSenderUsesLocalizedGroupKeyAndStableNonce(t *testing.T) {
	var nonces []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(request.URL.Path, "/users/@me/channels") {
			_, _ = writer.Write([]byte(`{"id":"dm"}`))
			return
		}
		var payload struct {
			Nonce        string            `json:"nonce"`
			EnforceNonce bool              `json:"enforce_nonce"`
			Components   []json.RawMessage `json:"components"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		nonces = append(nonces, payload.Nonce)
		if !payload.EnforceNonce || len(payload.Components) != 2 ||
			!strings.Contains(string(payload.Components[0]), "Miembro") ||
			strings.Contains(string(payload.Components[0]), "VERIFICARSE") {
			t.Errorf("payload = %#v %s", payload, payload.Components[0])
		}
		_, _ = writer.Write([]byte(`{"id":"message"}`))
	}))
	defer server.Close()
	originalUsers, originalChannels := discordgo.EndpointUsers, discordgo.EndpointChannels
	discordgo.EndpointUsers, discordgo.EndpointChannels = server.URL+"/users/", server.URL+"/channels/"
	t.Cleanup(func() {
		discordgo.EndpointUsers, discordgo.EndpointChannels = originalUsers, originalChannels
	})
	catalog, err := localization.Load(context.Background(), localization.Config{
		HTTPTimeout: time.Second, MaxBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	session, _ := discordgo.New("Bot test")
	client, err := discordplatform.New(discordplatform.Config{Token: "test", GuildID: "123"}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	client.SDK().Client = session.Client
	sender := NewSender(client, catalog)
	delivery := verificationnotification.Delivery{
		IdempotencyKey: "verification:verified:membership", Kind: verificationnotification.KindVerified,
		UserID: "user", GroupID: "group", GroupKey: "member",
	}
	for range 2 {
		messageID, sendErr := sender.Send(context.Background(), delivery)
		if sendErr != nil || messageID != "message" {
			t.Fatalf("Send() = %q, %v", messageID, sendErr)
		}
	}
	if len(nonces) != 2 || nonces[0] == "" || nonces[0] != nonces[1] {
		t.Fatalf("nonces = %#v", nonces)
	}
}

// TestSenderBuildsLocalizedUnverifiedMessage verifies explicit removal content.
func TestSenderBuildsLocalizedUnverifiedMessage(t *testing.T) {
	catalog, err := localization.Load(context.Background(), localization.Config{
		HTTPTimeout: time.Second, MaxBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	sender := &Sender{catalog: catalog}
	components, err := sender.components(verificationnotification.Delivery{
		Kind: verificationnotification.KindUnverified, GroupKey: "member",
	})
	if err != nil || len(components) != 1 {
		t.Fatalf("components() = %#v, %v", components, err)
	}
	text, ok := components[0].(discordgo.TextDisplay)
	if !ok || !strings.Contains(text.Content, "Miembro") {
		t.Fatalf("text = %#v", components[0])
	}
}

// TestSenderRejectsUnsupportedKind verifies invalid persisted events cannot be sent.
func TestSenderRejectsUnsupportedKind(t *testing.T) {
	catalog, err := localization.Load(context.Background(), localization.Config{
		HTTPTimeout: time.Second, MaxBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	sender := &Sender{catalog: catalog}
	if _, err = sender.components(verificationnotification.Delivery{Kind: "unknown", GroupKey: "member"}); err == nil {
		t.Fatal("components() error = nil")
	}
}

// TestSenderRejectsInvalidDiscordResponses verifies ambiguous delivery results are retried.
func TestSenderRejectsInvalidDiscordResponses(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "invalid JSON", response: `{`},
		{name: "missing message ID", response: `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(request.URL.Path, "/users/@me/channels") {
					_, _ = writer.Write([]byte(`{"id":"dm"}`))
					return
				}
				_, _ = writer.Write([]byte(test.response))
			}))
			defer server.Close()
			originalUsers, originalChannels := discordgo.EndpointUsers, discordgo.EndpointChannels
			discordgo.EndpointUsers, discordgo.EndpointChannels = server.URL+"/users/", server.URL+"/channels/"
			t.Cleanup(func() {
				discordgo.EndpointUsers, discordgo.EndpointChannels = originalUsers, originalChannels
			})
			catalog, err := localization.Load(context.Background(), localization.Config{
				HTTPTimeout: time.Second, MaxBytes: 1 << 20,
			})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			client, err := discordplatform.New(discordplatform.Config{Token: "test", GuildID: "123"}, zap.NewNop())
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			sender := provideSender(client, catalog)
			_, err = sender.Send(context.Background(), verificationnotification.Delivery{
				IdempotencyKey: "verification:unverified:membership", Kind: verificationnotification.KindUnverified,
				UserID: "user", GroupID: "group", GroupKey: "member",
			})
			if err == nil {
				t.Fatal("Send() error = nil")
			}
		})
	}
}
