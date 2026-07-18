package discord

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func TestGuildGatewayMemberCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("with_counts") != "true" {
			t.Errorf("query = %q, want with_counts=true", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"123","approximate_member_count":120,"approximate_presence_count":42}`))
	}))
	defer server.Close()
	original := discordgo.EndpointGuild
	discordgo.EndpointGuild = func(string) string { return server.URL }
	t.Cleanup(func() { discordgo.EndpointGuild = original })
	session, _ := discordgo.New("Bot test")
	gateway := NewGuildGateway(&Client{session: session, log: zap.NewNop(), guildID: "123"})
	memberCount, presenceCount, err := gateway.MemberCount(context.Background())
	if err != nil || memberCount != 120 || presenceCount != 42 {
		t.Fatalf("MemberCount() = %d, %d, error = %v", memberCount, presenceCount, err)
	}
}

func TestGuildGatewayMemberCountUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "upstream", http.StatusInternalServerError)
	}))
	defer server.Close()
	original := discordgo.EndpointGuild
	discordgo.EndpointGuild = func(string) string { return server.URL }
	t.Cleanup(func() { discordgo.EndpointGuild = original })
	session, _ := discordgo.New("Bot test")
	gateway := NewGuildGateway(&Client{session: session, log: zap.NewNop(), guildID: "123"})
	if _, _, err := gateway.MemberCount(context.Background()); err == nil {
		t.Fatal("MemberCount() error = nil")
	}
}
