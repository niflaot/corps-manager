package discord

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/inactivity"
	"github.com/niflaot/corps-manager/internal/messages"
	"go.uber.org/zap"
)

func TestMessageGatewayCreatesComponentsV2WithEnforcedNonce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			Nonce        string            `json:"nonce"`
			EnforceNonce bool              `json:"enforce_nonce"`
			Flags        int               `json:"flags"`
			Components   []json.RawMessage `json:"components"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Nonce != "stable-operation" || !payload.EnforceNonce || payload.Flags != int(discordgo.MessageFlagsIsComponentsV2) || len(payload.Components) != 1 {
			t.Errorf("payload = %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"789","guild_id":"123","channel_id":"456","flags":32768,"author":{"id":"bot"},"components":[{"type":10,"content":"Rules"}]}`))
	}))
	defer server.Close()
	original := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = original })
	session, _ := discordgo.New("Bot test")
	client := &Client{session: session, log: zap.NewNop(), guildID: "123"}
	client.userID.Store("bot")
	gateway := NewMessageGateway(client, "123")
	observed, err := gateway.Create(context.Background(), messages.CreateRequest{ChannelID: "456", Nonce: "stable-operation", Payload: v2Payload()})
	if err != nil || observed.ID != "789" || !observed.Owned || len(observed.Payload.Components) != 1 {
		t.Fatalf("Create() = %#v, error = %v", observed, err)
	}
}

func TestMessageGatewayRejectsConfiguredGuildMismatch(t *testing.T) {
	session, _ := discordgo.New("Bot test")
	gateway := NewMessageGateway(&Client{session: session, log: zap.NewNop(), guildID: "expected"}, "expected")
	if err := gateway.ValidateAssignment(context.Background(), "other", "456"); !errors.Is(err, messages.ErrInvalidAssignment) {
		t.Fatalf("error = %v", err)
	}
}

func TestMessageGatewayBlocksAmbiguousCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "upstream", http.StatusInternalServerError)
	}))
	defer server.Close()
	original := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = original })
	session, _ := discordgo.New("Bot test")
	gateway := NewMessageGateway(&Client{session: session, log: zap.NewNop(), guildID: "123"}, "123")
	_, err := gateway.Create(context.Background(), messages.CreateRequest{ChannelID: "456", Nonce: "stable", Payload: v2Payload()})
	if !errors.Is(err, messages.ErrAmbiguousCreate) {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestInactivityInteractionHelpers(t *testing.T) {
	data := discordgo.ModalSubmitInteractionData{Components: []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: inactivityNameInputID, Value: "Thomas_Jhonson"},
		}},
	}}
	if value := modalInput(data, inactivityNameInputID); value != "Thomas_Jhonson" {
		t.Fatalf("modalInput() = %q", value)
	}
	manager := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Member: &discordgo.Member{Permissions: discordgo.PermissionManageMessages},
	}}
	if !canManageRegistry(manager) || canManageRegistry(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{}}) {
		t.Fatal("inactivity registry permission check is incorrect")
	}
	page, update, ok := inactivityListPage(inactivityListPrefix + "2")
	if !ok || !update || page != 2 {
		t.Fatalf("inactivityListPage() = %d, %t, %t", page, update, ok)
	}
	content := renderInactivityList([]inactivity.Entry{{Name: "Thomas_Jhonson"}}, 21, 1, 2)
	if !strings.Contains(content, "Thomas_Jhonson") || !strings.Contains(content, "Página 2/2") {
		t.Fatalf("renderInactivityList() = %q", content)
	}
	if buttons := inactivityListButtons(0, 1); len(buttons) != 0 {
		t.Fatalf("single-page controls = %#v", buttons)
	}
}

func v2Payload() messages.Payload {
	return messages.Payload{Components: []messages.Component{messages.Component(`{"type":10,"content":"Rules"}`)}, AllowedMentions: messages.AllowedMentions{Parse: []string{}}}
}
