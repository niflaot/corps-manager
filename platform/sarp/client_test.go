package sarp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pixelados-net/discord-bot/internal/performance"
)

func TestClientFetchQueriesGtaRolBusiness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer tunnel-token" {
			t.Errorf("request method/auth = %s/%q", request.Method, request.Header.Get("Authorization"))
		}
		var body struct {
			Provider string `json:"provider"`
			Path     string `json:"path"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.Provider != "gta-rol" || body.Path != "/businesses/1995" {
			t.Errorf("request body = %#v, error = %v", body, err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"payload":{"id":1995,"name":"Benny","bank":10,"employees":[]}}`))
	}))
	defer server.Close()
	client := NewClient(performance.Config{Endpoint: server.URL, EndpointToken: "tunnel-token",
		HTTPTimeout: time.Second, MaxResponseBytes: 1024})
	snapshot, err := client.Fetch(context.Background(), 1995)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.BusinessID != 1995 || snapshot.Name != "Benny" || snapshot.Bank != 10 {
		t.Fatalf("Fetch() = %#v", snapshot)
	}
}
