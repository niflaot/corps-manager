package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"go.uber.org/zap"
)

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

func authenticatedRequest(method string, path string, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer test-api-key-long")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}
