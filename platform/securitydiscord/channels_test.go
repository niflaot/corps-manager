package securitydiscord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/security"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/zap"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (fn transportFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func testGateway(t *testing.T, respond func(*http.Request) (int, any)) *Gateway {
	t.Helper()
	previousLogger := discordgo.Logger
	t.Cleanup(func() { discordgo.Logger = previousLogger })
	client, err := discord.New(discord.Config{Token: "test", GuildID: "123"}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	client.SDK().Client = &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		code, value := respond(request)
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(encoded))), Request: request}, nil
	})}
	return NewGateway(client)
}

func TestEnsureTrapRepairsAssignmentAndRemovesDuplicates(t *testing.T) {
	channels := []*discordgo.Channel{
		{ID: "200", Name: security.TrapCategoryName, Type: discordgo.ChannelTypeGuildCategory, Position: 0},
		{ID: "300", Name: "Other", Type: discordgo.ChannelTypeGuildCategory, Position: 5},
		{ID: "400", Name: "renamed", Type: discordgo.ChannelTypeGuildText, ParentID: "300", NSFW: true, RateLimitPerUser: 20},
		{ID: "500", Name: security.TrapChannelName, Type: discordgo.ChannelTypeGuildText},
	}
	deleted := []string{}
	patched := map[string]map[string]any{}
	gateway := testGateway(t, func(request *http.Request) (int, any) {
		path := request.URL.Path
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(path, "/users/@me"):
			return 200, map[string]any{"id": "999"}
		case request.Method == http.MethodGet && strings.HasSuffix(path, "/guilds/123/channels"):
			return 200, channels
		case request.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			patched[path] = body
			return 200, map[string]any{"id": "400"}
		case request.Method == http.MethodDelete:
			deleted = append(deleted, path)
			return 200, map[string]any{}
		default:
			t.Errorf("unexpected request: %s %s", request.Method, path)
			return 400, map[string]any{}
		}
	})
	id, err := gateway.EnsureTrap(context.Background(), "400")
	if err != nil || id != "400" {
		t.Fatalf("EnsureTrap=%q, %v", id, err)
	}
	if len(deleted) != 1 || !strings.HasSuffix(deleted[0], "/channels/500") {
		t.Fatalf("deleted=%v", deleted)
	}
	for path, body := range patched {
		if strings.HasSuffix(path, "/channels/400") {
			if body["name"] != security.TrapChannelName || body["parent_id"] != "200" || body["topic"] != security.TrapWarning || body["nsfw"] != false || body["rate_limit_per_user"] != float64(0) {
				t.Fatalf("trap repair=%v", body)
			}
		}
		if strings.HasSuffix(path, "/channels/200") && body["position"] != float64(6) {
			t.Fatalf("category position=%v", body)
		}
	}
	if len(patched) != 2 {
		t.Fatalf("repairs=%v", patched)
	}
}

func TestHealthyTrapProducesNoMutations(t *testing.T) {
	channels := []*discordgo.Channel{
		{ID: "200", Name: security.TrapCategoryName, Type: discordgo.ChannelTypeGuildCategory, Position: 5},
		{ID: "400", Name: security.TrapChannelName, Type: discordgo.ChannelTypeGuildText, ParentID: "200", Topic: security.TrapWarning, PermissionOverwrites: trapPermissions("123", "999")},
	}
	gateway := testGateway(t, func(request *http.Request) (int, any) {
		if request.Method != http.MethodGet {
			t.Errorf("unexpected mutation %s", request.Method)
		}
		if strings.HasSuffix(request.URL.Path, "/users/@me") {
			return 200, map[string]any{"id": "999"}
		}
		return 200, channels
	})
	for range 2 {
		if _, err := gateway.EnsureTrap(context.Background(), "400"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRoleAndBanUseGuildScopedIdempotentEndpoints(t *testing.T) {
	paths := []string{}
	gateway := testGateway(t, func(request *http.Request) (int, any) {
		paths = append(paths, fmt.Sprintf("%s %s", request.Method, request.URL.Path))
		if request.Method == http.MethodGet {
			return 200, []map[string]any{{"id": "456", "managed": false}, {"id": "789", "managed": true}}
		}
		return 204, nil
	})
	if err := gateway.GrantRole(context.Background(), "111", "456"); err != nil {
		t.Fatal(err)
	}
	if err := gateway.GrantRole(context.Background(), "111", "789"); err == nil {
		t.Fatal("managed role accepted")
	}
	if err := gateway.GrantRole(context.Background(), "111", "000"); err == nil {
		t.Fatal("missing role accepted")
	}
	if err := gateway.Ban(context.Background(), "111"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(paths, "\n")
	if !strings.Contains(joined, "PUT /api/v9/guilds/123/members/111/roles/456") || !strings.Contains(joined, "PUT /api/v9/guilds/123/bans/111") {
		t.Fatalf("requests=%s", joined)
	}
}
