package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestBusPublishesAndSubscribes(t *testing.T) {
	bus := New(context.Background(), zap.NewNop())
	t.Cleanup(bus.Close)
	received := make(chan Event, 1)
	unsubscribe, err := bus.Subscribe(context.Background(), "messages.reconciled", func(_ context.Context, event Event) error {
		received <- event
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	t.Cleanup(unsubscribe)
	id, err := bus.Publish("messages.reconciled", "rules")
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	select {
	case event := <-received:
		if event.ID != id || event.Name != "messages.reconciled" || event.Payload != "rules" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}
}

func TestBusValidatesNamesAndStops(t *testing.T) {
	bus := New(context.Background(), zap.NewNop())
	if _, err := bus.Publish("Invalid Event", nil); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("invalid Publish() error = %v", err)
	}
	bus.Close()
	if _, err := bus.Publish("application.stopped", nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("closed Publish() error = %v", err)
	}
}
