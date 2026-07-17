package openapi

import (
	"encoding/json"
	"testing"
)

func TestSpecIsJSON(t *testing.T) {
	var document map[string]any
	if err := json.Unmarshal([]byte(Spec), &document); err != nil {
		t.Fatalf("Spec is invalid JSON: %v", err)
	}
}
