package discord

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestDisabledClientStopsWithContext(t *testing.T) {
	client, err := New(Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.Enabled() || client.SDK() != nil {
		t.Fatal("client should be disabled")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
