package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/messages"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"go.uber.org/zap"
)

type messageServiceStub struct {
	created   messages.Definition
	reconcile string
}

func (service *messageServiceStub) Create(_ context.Context, definition messages.Definition, _ string) (messages.MutationResult, error) {
	service.created = definition
	return messages.MutationResult{Record: messages.Record{Definition: definition, ID: "id", Revision: 1, State: messages.StatePending}}, nil
}

func (*messageServiceStub) Get(context.Context, string) (messages.Record, error) {
	return messages.Record{}, messages.ErrNotFound
}

func (*messageServiceStub) List(context.Context, messages.ListQuery) (messages.Page, error) {
	return messages.Page{Items: []messages.Record{}}, nil
}

func (*messageServiceStub) Replace(context.Context, string, uint64, messages.Definition, string) (messages.MutationResult, error) {
	return messages.MutationResult{}, messages.ErrConflict
}

func (*messageServiceStub) Archive(context.Context, string, uint64, string) (messages.MutationResult, error) {
	return messages.MutationResult{}, messages.ErrConflict
}

func (service *messageServiceStub) Reconcile(_ context.Context, key string) error {
	service.reconcile = key
	return nil
}

func TestMessageRoutesRequireAuthenticationAndCreate(t *testing.T) {
	service := &messageServiceStub{}
	application := messageTestApplication(service)
	body := []byte(`{"key":"rules","guildId":"123","channelId":"456","payload":{"components":[{"type":10,"content":"Rules"}],"allowedMentions":{"parse":[]}}}`)
	unauthorized := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
	unauthorized.Header.Set("Content-Type", "application/json")
	response, err := application.Test(unauthorized)
	if err != nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, error = %v", response.StatusCode, err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	request.Header.Set("Idempotency-Key", "request-1")
	response, err = application.Test(request)
	if err != nil || response.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, error = %v", response.StatusCode, err)
	}
	if service.created.Key != "rules" {
		t.Fatalf("created = %#v", service.created)
	}
}

func TestMessageRoutesRejectUnknownJSONAndTriggerReconcile(t *testing.T) {
	service := &messageServiceStub{}
	application := messageTestApplication(service)
	request := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewBufferString(`{"unknown":true}`))
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	request.Header.Set("Idempotency-Key", "request-1")
	response, err := application.Test(request)
	if err != nil || response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, error = %v", response.StatusCode, err)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/messages/rules/reconcile", nil)
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	response, err = application.Test(request)
	if err != nil || response.StatusCode != http.StatusAccepted || service.reconcile != "rules" {
		t.Fatalf("reconcile status = %d, key = %q, error = %v", response.StatusCode, service.reconcile, err)
	}
}

func TestMessageResponseJSON(t *testing.T) {
	record := messages.Record{Definition: messages.Definition{Key: "rules"}, ID: "id", Revision: 1}
	encoded, err := json.Marshal(record)
	if err != nil || !bytes.Contains(encoded, []byte(`"revision":1`)) {
		t.Fatalf("Marshal() = %s, error = %v", encoded, err)
	}
}

func messageTestApplication(service MessageService) *fiber.App {
	healthService := health.New(map[string]health.Check{})
	return New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest}, Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, healthService, Dependencies{Messages: service}, "1.0.0")
}
