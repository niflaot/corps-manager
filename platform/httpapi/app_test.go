package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/niflaot/corps-manager/internal/inactivity"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"go.uber.org/zap"
)

type inactivityHTTPStub struct {
	items   []inactivity.Entry
	removed string
}

func (stub *inactivityHTTPStub) List(context.Context) ([]inactivity.Entry, error) {
	return stub.items, nil
}

func (stub *inactivityHTTPStub) Add(_ context.Context, name string, actor string) (inactivity.Entry, error) {
	entry := inactivity.Entry{Name: name, AddedBy: actor}
	stub.items = append(stub.items, entry)
	return entry, nil
}

func (stub *inactivityHTTPStub) Remove(_ context.Context, name string) error {
	stub.removed = name
	return nil
}

func TestStatus(t *testing.T) {
	healthService := health.New(map[string]health.Check{
		"postgres": func(context.Context) error { return nil },
		"discord":  func(context.Context) error { return errors.New("offline") },
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
	if status.Dependencies["postgres"] != health.StatusAvailable || status.Dependencies["discord"] != health.StatusUnavailable {
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

func TestInactivityRoutesRequireAuthenticationAndMutateRegistry(t *testing.T) {
	stub := &inactivityHTTPStub{}
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest},
		Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil),
		Dependencies{Inactivity: stub}, "1.0.0")

	unauthorized, err := application.Test(httptest.NewRequest(http.MethodGet, "/api/inactivity", nil))
	if err != nil || unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, error = %v", unauthorized.StatusCode, err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/inactivity", bytes.NewBufferString(`{"name":"Thomas_Jhonson"}`))
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	request.Header.Set("Content-Type", "application/json")
	created, err := application.Test(request)
	if err != nil || created.StatusCode != http.StatusCreated || len(stub.items) != 1 || stub.items[0].AddedBy != "api" {
		t.Fatalf("create status = %d, items = %#v, error = %v", created.StatusCode, stub.items, err)
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/inactivity/Thomas_Jhonson", nil)
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	deleted, err := application.Test(request)
	if err != nil || deleted.StatusCode != http.StatusNoContent || stub.removed != "Thomas_Jhonson" {
		t.Fatalf("delete status = %d, removed = %q, error = %v", deleted.StatusCode, stub.removed, err)
	}
}
