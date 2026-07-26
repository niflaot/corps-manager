package notification

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

type serviceRepository struct {
	Repository
	// inserted controls whether persistence creates a new row.
	inserted bool
	// err is the configured persistence failure.
	err error
	// events records attempted publications.
	events []Event
}

// Enqueue records one test publication.
func (repository *serviceRepository) Enqueue(_ context.Context, event Event) (bool, error) {
	repository.events = append(repository.events, event)
	return repository.inserted, repository.err
}

// TestServiceSignalsOnlyNewValidEvents verifies local wakeup coalescing.
func TestServiceSignalsOnlyNewValidEvents(t *testing.T) {
	repository := &serviceRepository{inserted: true}
	signal := NewSignal()
	service := NewService(repository, signal, zap.NewNop())
	event := NewEvent(KindVerified, "membership", "user", "group", "member")
	service.Enqueue(context.Background(), event)
	select {
	case <-signal.C():
	default:
		t.Fatal("notification signal was not emitted")
	}
	repository.inserted = false
	service.Enqueue(context.Background(), event)
	select {
	case <-signal.C():
		t.Fatal("duplicate notification emitted a signal")
	default:
	}
	if len(repository.events) != 2 || repository.events[0].IdempotencyKey != "verification:verified:membership" {
		t.Fatalf("events = %#v", repository.events)
	}
}

// TestServiceRejectsInvalidEventsWithoutPersistence verifies publication validation.
func TestServiceRejectsInvalidEventsWithoutPersistence(t *testing.T) {
	repository := &serviceRepository{inserted: true}
	service := NewService(repository, NewSignal(), zap.NewNop())
	service.Enqueue(context.Background(), Event{Kind: KindVerified})
	if len(repository.events) != 0 {
		t.Fatalf("events = %#v", repository.events)
	}
}

// TestServiceKeepsPersistenceFailuresInformational verifies non-fatal enqueue behavior.
func TestServiceKeepsPersistenceFailuresInformational(t *testing.T) {
	repository := &serviceRepository{err: errors.New("database unavailable")}
	service := NewService(repository, NewSignal(), zap.NewNop())
	service.Enqueue(context.Background(), NewEvent(KindVerified, "membership", "user", "group", "member"))
	if len(repository.events) != 1 {
		t.Fatalf("events = %#v", repository.events)
	}
}
