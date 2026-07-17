package redis

import (
	"context"
	"testing"
	"time"
)

func TestNewMapsConfig(t *testing.T) {
	client := New(Config{Address: "127.0.0.1:6380", Database: 2})
	t.Cleanup(func() { _ = client.Close() })
	options := client.SDK().Options()
	if options.Addr != "127.0.0.1:6380" || options.DB != 2 {
		t.Fatalf("Options() = %#v", options)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx); err == nil {
		t.Fatal("Ping() error = nil for unavailable endpoint")
	}
}
