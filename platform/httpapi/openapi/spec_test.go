package openapi

import (
	"encoding/json"
	"testing"
)

type testOperation struct {
	Security []map[string]any `json:"security"`
}

type testPathItem struct {
	Get    testOperation `json:"get"`
	Post   testOperation `json:"post"`
	Delete testOperation `json:"delete"`
}

func TestSpecIsJSON(t *testing.T) {
	var document map[string]any
	if err := json.Unmarshal([]byte(Spec), &document); err != nil {
		t.Fatalf("Spec is invalid JSON: %v", err)
	}
}

func TestSpecDocumentsDiscordLinkSecurityBoundary(t *testing.T) {
	var document struct {
		Paths map[string]testPathItem `json:"paths"`
	}
	if err := json.Unmarshal([]byte(Spec), &document); err != nil {
		t.Fatalf("Spec is invalid JSON: %v", err)
	}
	publicOperations := [][2]string{{"/oauth/discord/start/{intentId}", "get"},
		{"/oauth/discord/callback", "get"}}
	for _, operation := range publicOperations {
		path, ok := document.Paths[operation[0]]
		if !ok {
			t.Fatalf("missing public operation %s %s", operation[1], operation[0])
		}
		if len(selectOperation(path, operation[1]).Security) != 0 {
			t.Fatalf("public operation %s unexpectedly requires API authentication", operation[0])
		}
	}
	privateOperations := [][2]string{{"/api/discord-links/intents", "post"},
		{"/api/discord-links/login-intents", "post"},
		{"/api/discord-links/results/exchange", "post"},
		{"/api/discord-links/subjects/{subject}", "get"},
		{"/api/discord-links/subjects/{subject}", "delete"},
		{"/api/discord-links/users/{discordUserId}", "get"}}
	for _, operation := range privateOperations {
		if len(selectOperation(document.Paths[operation[0]], operation[1]).Security) == 0 {
			t.Fatalf("private operation %s %s does not require API authentication", operation[1], operation[0])
		}
	}
}

func selectOperation(path testPathItem, method string) testOperation {
	switch method {
	case "post":
		return path.Post
	case "delete":
		return path.Delete
	default:
		return path.Get
	}
}
