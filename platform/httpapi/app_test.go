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

func TestStatusAndDocumentation(t *testing.T) {
	healthService := health.New(map[string]health.Check{
		"postgres": func(context.Context) error { return nil },
		"redis":    func(context.Context) error { return errors.New("offline") },
	})
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest}, Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, healthService, Dependencies{}, "1.0.0")
	response, err := application.Test(httptest.NewRequest("GET", "/status", nil))
	if err != nil {
		t.Fatalf("status request: %v", err)
	}
	var status StatusResponse
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Dependencies["postgres"] != health.StatusAvailable || status.Dependencies["redis"] != health.StatusUnavailable {
		t.Fatalf("status = %#v", status)
	}
	docs, err := application.Test(httptest.NewRequest("GET", "/docs", nil))
	if err != nil || docs.StatusCode != 200 {
		t.Fatalf("docs status = %d, error = %v", docs.StatusCode, err)
	}
}
