package discordoauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestClientCompletesAndRevokesAuthorizationCodeGrant(t *testing.T) {
	requests := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests <- request.URL.Path
		if request.UserAgent() != userAgent {
			t.Errorf("User-Agent = %q", request.UserAgent())
		}
		switch request.URL.Path {
		case "/oauth2/token":
			username, password, ok := request.BasicAuth()
			if !ok || username != "123" || password != "secret" || request.FormValue("code") != "code" {
				t.Errorf("token credentials or form are invalid")
			}
			_ = json.NewEncoder(response).Encode(map[string]string{
				"access_token": "access", "token_type": "Bearer", "scope": "identify",
			})
		case "/users/@me":
			if request.Header.Get("Authorization") != "Bearer access" {
				t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(response).Encode(map[string]any{
				"id": "456", "username": "tester", "global_name": "Test User", "avatar": "hash",
			})
		case "/oauth2/token/revoke":
			if request.FormValue("token") != "access" {
				t.Errorf("revoked token = %q", request.FormValue("token"))
			}
			response.WriteHeader(http.StatusNoContent)
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	config := Config{Enabled: true, ClientID: "123", ClientSecret: "secret",
		PublicURL: "https://public.example.test", HTTPTimeout: time.Second}
	client := New(config, zap.NewNop())
	client.apiBase = server.URL
	client.authorizeURL = server.URL + "/authorize"
	authorization, err := url.Parse(client.AuthorizationURL("state"))
	if err != nil || authorization.Query().Get("scope") != "identify" ||
		authorization.Query().Get("redirect_uri") != config.CallbackURL() {
		t.Fatalf("AuthorizationURL() = %v, error = %v", authorization, err)
	}
	grant, err := client.Exchange(context.Background(), "code")
	if err != nil || grant.AccessToken != "access" {
		t.Fatalf("Exchange() = %#v, error = %v", grant, err)
	}
	identity, err := client.CurrentUser(context.Background(), grant)
	if err != nil || identity.UserID != "456" || identity.GlobalName != "Test User" {
		t.Fatalf("CurrentUser() = %#v, error = %v", identity, err)
	}
	if err = client.Revoke(context.Background(), grant); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	for _, expected := range []string{"/oauth2/token", "/users/@me", "/oauth2/token/revoke"} {
		if actual := <-requests; actual != expected {
			t.Fatalf("request path = %q, want %q", actual, expected)
		}
	}
}

func TestClientMapsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	client := New(Config{Enabled: true, ClientID: "123", ClientSecret: "secret",
		PublicURL: "https://public.example.test", HTTPTimeout: time.Second}, zap.NewNop())
	client.apiBase = server.URL
	if _, err := client.Exchange(context.Background(), "code"); err == nil {
		t.Fatal("Exchange() error = nil")
	}
}
