package discord

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"go.uber.org/zap"
)

func TestMessageGatewayCreatesWithEnforcedNonce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/channels/456/messages" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		var payload struct {
			Nonce        string `json:"nonce"`
			EnforceNonce bool   `json:"enforce_nonce"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Nonce != "stable-operation" || !payload.EnforceNonce {
			t.Errorf("payload = %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"789","channel_id":"456","content":"Rules","author":{"id":"bot"},"embeds":[]}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	session, err := discordgo.New("Bot test")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}
	client := &Client{session: session, log: zap.NewNop()}
	client.userID.Store("bot")
	gateway := NewMessageGateway(client)
	observed, err := gateway.Create(context.Background(), messages.CreateRequest{
		ChannelID: "456", Nonce: "stable-operation",
		Payload: messages.Payload{Content: "Rules", Embeds: []messages.Embed{}, AllowedMentions: messages.AllowedMentions{Parse: []string{}}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if observed.ID != "789" || !observed.Owned || observed.Payload.Content != "Rules" {
		t.Fatalf("Create() = %#v", observed)
	}
}

func TestMessageGatewayRejectsGuildMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"456","guild_id":"other"}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	session, err := discordgo.New("Bot test")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}
	gateway := NewMessageGateway(&Client{session: session, log: zap.NewNop()})
	if err := gateway.ValidateAssignment(context.Background(), "expected", "456"); !errors.Is(err, messages.ErrInvalidAssignment) {
		t.Fatalf("ValidateAssignment() error = %v", err)
	}
}

func TestMessageGatewayBlocksAmbiguousCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "upstream failure", http.StatusInternalServerError)
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	session, err := discordgo.New("Bot test")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}
	gateway := NewMessageGateway(&Client{session: session, log: zap.NewNop()})
	_, err = gateway.Create(context.Background(), messages.CreateRequest{ChannelID: "456", Nonce: "stable", Payload: messages.Payload{Content: "Rules"}})
	if !errors.Is(err, messages.ErrAmbiguousCreate) {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestDiscordEmbedRoundTripIgnoresManagedResponseFields(t *testing.T) {
	payload := []messages.Embed{{
		Title: "Title", Description: "Body", URL: "https://example.com", Color: 42,
		Author: &messages.EmbedAuthor{Name: "Author", IconURL: "https://example.com/author.png"},
		Footer: &messages.EmbedFooter{Text: "Footer"}, Image: &messages.EmbedMedia{URL: "https://example.com/image.png"},
		Fields: []messages.EmbedField{{Name: "Name", Value: "Value", Inline: true}},
	}}
	roundTrip := fromDiscordEmbeds(toDiscordEmbeds(payload))
	if len(roundTrip) != 1 || roundTrip[0].Title != payload[0].Title || len(roundTrip[0].Fields) != 1 {
		t.Fatalf("round trip = %#v", roundTrip)
	}
}
