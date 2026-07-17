package health

import (
	"context"
	"errors"
	"testing"
)

func TestSnapshot(t *testing.T) {
	service := New(map[string]Check{
		"available":   func(context.Context) error { return nil },
		"unavailable": func(context.Context) error { return errors.New("offline") },
	})
	statuses := service.Snapshot(context.Background())
	if statuses["available"] != StatusAvailable || statuses["unavailable"] != StatusUnavailable {
		t.Fatalf("Snapshot() = %#v", statuses)
	}
}
