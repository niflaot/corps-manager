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

func TestSpecProtectsPrivateOperations(t *testing.T) {
	var document struct {
		Paths map[string]testPathItem `json:"paths"`
	}
	if err := json.Unmarshal([]byte(Spec), &document); err != nil {
		t.Fatalf("Spec is invalid JSON: %v", err)
	}
	privateOperations := [][2]string{{"/api/performance", "get"},
		{"/api/performance/refresh", "post"}, {"/api/inactivity", "get"}, {"/api/inactivity", "post"},
		{"/api/inactivity/{name}", "delete"}, {"/api/messages", "get"}, {"/api/messages", "post"}}
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
