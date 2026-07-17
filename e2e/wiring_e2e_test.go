package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pixelados-net/discord-bot/platform/health"
	"github.com/pixelados-net/discord-bot/platform/httpapi"
)

func TestBaseWiringE2E(t *testing.T) {
	runtime := startHarness(t)
	response, err := runtime.client.Get(runtime.baseURL + "/status")
	if err != nil {
		t.Fatalf("GET /status: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d", response.StatusCode)
	}
	var status httpapi.StatusResponse
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Version != "1.0.0" || status.Dependencies["discord"] != health.StatusDisabled {
		t.Fatalf("status = %#v", status)
	}
}
