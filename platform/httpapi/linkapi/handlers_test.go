package linkapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

type routeService struct {
	intent discordlinks.CreateIntent
}

func (service *routeService) CreateIntent(_ context.Context,
	request discordlinks.CreateIntent) (discordlinks.Intent, error) {
	service.intent = request
	return discordlinks.Intent{ID: "00000000-0000-0000-0000-000000000001",
		Kind: discordlinks.IntentKindLink, Subject: request.Subject, CompletionKey: request.CompletionKey}, nil
}

func (*routeService) CreateLoginIntent(context.Context,
	discordlinks.CreateLoginIntent) (discordlinks.Intent, error) {
	return discordlinks.Intent{}, nil
}

func (*routeService) Start(context.Context, string) (string, error) {
	return "https://discord.example/authorize", nil
}

func (*routeService) Complete(context.Context, discordlinks.Callback) (discordlinks.Completion, error) {
	return discordlinks.Completion{CompletionKey: "links", Code: "result-code"}, nil
}

func (*routeService) ExchangeResult(context.Context, string, string) (discordlinks.Result, error) {
	return discordlinks.Result{}, nil
}

func (*routeService) LinkBySubject(context.Context, string) (discordlinks.Link, error) {
	return discordlinks.Link{}, nil
}

func (*routeService) LinkByDiscordUser(context.Context, string) (discordlinks.Link, error) {
	return discordlinks.Link{}, nil
}

func (*routeService) Unlink(context.Context, string) (discordlinks.Link, error) {
	return discordlinks.Link{}, nil
}

func TestRoutesCreateIntentAndRedirectWithoutAcceptingReturnURL(t *testing.T) {
	service := &routeService{}
	routes := New(service, Config{publicURL: "https://discord-api.example.test",
		CompletionURLs: map[string]string{"links": "https://pixelados.example.test/result"}})
	application := fiber.New()
	routes.RegisterPublic(application)
	routes.RegisterPrivate(application.Group("/api/discord-links"))
	request := httptest.NewRequest(http.MethodPost, "/api/discord-links/intents",
		bytes.NewBufferString(`{"subject":"user:test-001","completionKey":"links"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "link-test-001")
	response, err := application.Test(request)
	if err != nil || response.StatusCode != http.StatusCreated || service.intent.Subject != "user:test-001" ||
		service.intent.IdempotencyKey != "link-test-001" {
		t.Fatalf("create status = %d, intent = %#v, error = %v", response.StatusCode, service.intent, err)
	}
	callback := httptest.NewRequest(http.MethodGet, "/oauth/discord/callback?state=state&code=code", nil)
	response, err = application.Test(callback)
	if err != nil || response.StatusCode != http.StatusSeeOther ||
		response.Header.Get("Location") != "https://pixelados.example.test/result?code=result-code" {
		t.Fatalf("callback status = %d, location = %q, error = %v", response.StatusCode,
			response.Header.Get("Location"), err)
	}
}
