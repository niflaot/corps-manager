package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"go.uber.org/zap"
)

type guildServiceStub struct {
	memberCount, presenceCount int
	err                        error
}

func (service *guildServiceStub) MemberCount(context.Context) (int, int, error) {
	return service.memberCount, service.presenceCount, service.err
}

func TestGuildMemberCountRoute(t *testing.T) {
	service := &guildServiceStub{memberCount: 120, presenceCount: 42}
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest},
		Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil), Dependencies{Guild: service}, "1.0.0")
	response, err := application.Test(httptest.NewRequest("GET", "/guild/members", nil))
	if err != nil {
		t.Fatalf("guild members request: %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var stats GuildStats
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		t.Fatalf("decode guild stats: %v", err)
	}
	if stats.MemberCount != 120 || stats.PresenceCount != 42 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestGuildMemberCountRouteUnavailable(t *testing.T) {
	service := &guildServiceStub{err: errors.New("discord unavailable")}
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest},
		Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil), Dependencies{Guild: service}, "1.0.0")
	response, err := application.Test(httptest.NewRequest("GET", "/guild/members", nil))
	if err != nil {
		t.Fatalf("guild members request: %v", err)
	}
	if response.StatusCode != 503 {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestGuildMemberCountRouteAbsentWhenUnconfigured(t *testing.T) {
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest},
		Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil), Dependencies{}, "1.0.0")
	response, err := application.Test(httptest.NewRequest("GET", "/guild/members", nil))
	if err != nil {
		t.Fatalf("guild members request: %v", err)
	}
	if response.StatusCode != 404 {
		t.Fatalf("status = %d", response.StatusCode)
	}
}
