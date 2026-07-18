package messages

import (
	"context"
	"testing"
)

type serviceRepository struct {
	Repository
	created Definition
	due     string
}

func (repository *serviceRepository) Create(_ context.Context, definition Definition, _ Idempotency) (MutationResult, error) {
	repository.created = definition
	return MutationResult{Record: Record{Definition: definition, Revision: 1}}, nil
}

func (repository *serviceRepository) MarkDue(_ context.Context, key string) error {
	repository.due = key
	return nil
}

func TestServiceCreatesAndTriggers(t *testing.T) {
	repository := &serviceRepository{}
	signal := NewSignal()
	service := NewService(repository, signal)
	result, err := service.Create(context.Background(), validDefinition(), "request-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Record.Revision != 1 || repository.created.Payload.Components == nil {
		t.Fatalf("Create() = %#v", result)
	}
	select {
	case <-signal.C():
	default:
		t.Fatal("create did not trigger reconciliation")
	}
}

func TestServiceRequiresIdempotencyAndMarksDue(t *testing.T) {
	repository := &serviceRepository{}
	signal := NewSignal()
	service := NewService(repository, signal)
	if _, err := service.Create(context.Background(), validDefinition(), ""); err == nil {
		t.Fatal("Create() error = nil")
	}
	if err := service.Reconcile(context.Background(), "rules"); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if repository.due != "rules" {
		t.Fatalf("due key = %q", repository.due)
	}
}

func TestServiceRejectsInvalidListFilters(t *testing.T) {
	service := NewService(&serviceRepository{}, NewSignal())
	for _, query := range []ListQuery{{Limit: -1}, {Offset: -1}, {State: "unknown"}} {
		if _, err := service.List(context.Background(), query); err == nil {
			t.Fatalf("List(%#v) error = nil", query)
		}
	}
}
