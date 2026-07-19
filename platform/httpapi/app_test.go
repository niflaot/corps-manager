package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"go.uber.org/zap"
)

type linkRoutesStub struct{}

func (linkRoutesStub) RegisterPublic(router fiber.Router) {
	router.Get("/oauth/test", func(ctx *fiber.Ctx) error { return ctx.SendStatus(204) })
}

func (linkRoutesStub) RegisterPrivate(router fiber.Router) {
	router.Get("/test", func(ctx *fiber.Ctx) error { return ctx.SendStatus(204) })
}

func TestStatus(t *testing.T) {
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
}

func TestDocumentationRoutesAreDevelopmentOnly(t *testing.T) {
	environments := []struct {
		name        string
		environment appconfig.Environment
		status      int
	}{
		{name: "development", environment: appconfig.EnvironmentDevelopment, status: 200},
		{name: "test", environment: appconfig.EnvironmentTest, status: 404},
		{name: "production", environment: appconfig.EnvironmentProduction, status: 404},
	}
	for _, item := range environments {
		t.Run(item.name, func(t *testing.T) {
			application := New(zap.NewNop(), appconfig.Config{Environment: item.environment},
				Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil), Dependencies{}, "1.0.0")
			for _, path := range []string{"/docs", "/openapi.json"} {
				response, err := application.Test(httptest.NewRequest("GET", path, nil))
				if err != nil || response.StatusCode != item.status {
					t.Fatalf("%s status = %d, error = %v", path, response.StatusCode, err)
				}
			}
		})
	}
}

func TestDiscordLinkBrowserRoutesArePublicAndServiceRoutesArePrivate(t *testing.T) {
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest},
		Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil),
		Dependencies{DiscordLinks: linkRoutesStub{}}, "1.0.0")
	publicResponse, err := application.Test(httptest.NewRequest("GET", "/oauth/test", nil))
	if err != nil || publicResponse.StatusCode != 204 {
		t.Fatalf("public status = %d, error = %v", publicResponse.StatusCode, err)
	}
	privateResponse, err := application.Test(httptest.NewRequest("GET", "/api/discord-links/test", nil))
	if err != nil || privateResponse.StatusCode != 401 {
		t.Fatalf("private status = %d, error = %v", privateResponse.StatusCode, err)
	}
	request := httptest.NewRequest("GET", "/api/discord-links/test", nil)
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	privateResponse, err = application.Test(request)
	if err != nil || privateResponse.StatusCode != 204 {
		t.Fatalf("authenticated status = %d, error = %v", privateResponse.StatusCode, err)
	}
}
