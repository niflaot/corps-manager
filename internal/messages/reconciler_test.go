package messages

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type reconcileRepository struct {
	Repository
	record      Record
	completion  Completion
	release     Release
	completeErr error
	releaseErr  error
}

func (repository *reconcileRepository) ClaimDue(context.Context, ClaimRequest) ([]Record, error) {
	return []Record{repository.record}, nil
}

func (repository *reconcileRepository) Complete(_ context.Context, completion Completion) error {
	repository.completion = completion
	return repository.completeErr
}

func (repository *reconcileRepository) Release(_ context.Context, release Release) error {
	repository.release = release
	return repository.releaseErr
}

type reconcileGateway struct {
	Gateway
	observed        ObservedMessage
	getError        error
	assignmentError error
	createObserved  *ObservedMessage
	createCalls     int
	replaceCalls    int
	deleteCalls     int
	nonce           string
}

func (gateway *reconcileGateway) ValidateAssignment(context.Context, string, string) error {
	return gateway.assignmentError
}

func (gateway *reconcileGateway) Get(context.Context, string, string) (ObservedMessage, error) {
	return gateway.observed, gateway.getError
}

func (gateway *reconcileGateway) Create(_ context.Context, request CreateRequest) (ObservedMessage, error) {
	gateway.createCalls++
	gateway.nonce = request.Nonce
	if gateway.createObserved != nil {
		return *gateway.createObserved, nil
	}
	return ObservedMessage{ID: "new", ChannelID: request.ChannelID, Payload: request.Payload, Owned: true, ComponentsV2: true}, nil
}

func (gateway *reconcileGateway) Replace(_ context.Context, request ReplaceRequest) (ObservedMessage, error) {
	gateway.replaceCalls++
	return ObservedMessage{ID: request.MessageID, ChannelID: request.ChannelID, Payload: request.Payload, Owned: true, ComponentsV2: true}, nil
}

func (gateway *reconcileGateway) Delete(context.Context, string, string) error {
	gateway.deleteCalls++
	return nil
}

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

func TestReconcilerRepairsDrift(t *testing.T) {
	record := reconciliationRecord(t)
	gateway := &reconcileGateway{observed: ObservedMessage{ID: "remote", ChannelID: record.ChannelID, Payload: Payload{Components: []Component{Component(`{"type":10,"content":"damaged"}`)}, AllowedMentions: AllowedMentions{Parse: []string{}}}, Owned: true}}
	repository := &reconcileRepository{record: record}
	reconciler := NewReconciler(repository, gateway, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); err != nil {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if gateway.replaceCalls != 1 || !repository.completion.Repaired || repository.completion.ObservedHash != record.DesiredHash {
		t.Fatalf("replace calls = %d, completion = %#v", gateway.replaceCalls, repository.completion)
	}
}

func TestReconcilerUpgradesMatchingLegacyComponentsToV2(t *testing.T) {
	record := reconciliationRecord(t)
	gateway := &reconcileGateway{observed: ObservedMessage{ID: "remote", ChannelID: record.ChannelID, Payload: record.Payload, Owned: true}}
	repository := &reconcileRepository{record: record}
	reconciler := NewReconciler(repository, gateway, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); err != nil {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if gateway.replaceCalls != 1 || !repository.completion.Repaired {
		t.Fatalf("replace calls = %d, completion = %#v", gateway.replaceCalls, repository.completion)
	}
}

func TestReconcilerRecreatesMissingMessageWithStableNonce(t *testing.T) {
	record := reconciliationRecord(t)
	gateway := &reconcileGateway{getError: ErrNotFound}
	repository := &reconcileRepository{record: record}
	reconciler := NewReconciler(repository, gateway, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); err != nil {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if gateway.createCalls != 1 || len(gateway.nonce) != 25 || repository.completion.DiscordMessageID != "new" {
		t.Fatalf("create calls = %d, nonce = %q, completion = %#v", gateway.createCalls, gateway.nonce, repository.completion)
	}
}

func TestReconcilerPreservesCreatedMessageIDWhenDiscordResponseDrifts(t *testing.T) {
	record := reconciliationRecord(t)
	record.DiscordMessageID = ""
	gatewayCreatePayload := record.Payload
	gatewayCreatePayload.Components[0] = Component(`{"type":10,"content":"changed"}`)
	gateway := &reconcileGateway{createObserved: &ObservedMessage{ID: "new", ChannelID: record.ChannelID,
		Payload: gatewayCreatePayload, Owned: true, ComponentsV2: true}}
	repository := &reconcileRepository{record: record}
	reconciler := NewReconciler(repository, gateway, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); err != nil {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if repository.release.DiscordMessageID != "new" {
		t.Fatalf("release = %#v", repository.release)
	}
}

func TestReconcilerDeletesCreatedOrphanAfterRevisionConflict(t *testing.T) {
	record := reconciliationRecord(t)
	record.DiscordMessageID = ""
	repository := &reconcileRepository{record: record, completeErr: ErrConflict}
	gateway := &reconcileGateway{}
	reconciler := NewReconciler(repository, gateway, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if gateway.createCalls != 1 || gateway.deleteCalls != 1 {
		t.Fatalf("create calls = %d, delete calls = %d", gateway.createCalls, gateway.deleteCalls)
	}
}

func TestReconcilerBlocksForeignMessage(t *testing.T) {
	record := reconciliationRecord(t)
	gateway := &reconcileGateway{observed: ObservedMessage{ID: "remote", Owned: false}}
	repository := &reconcileRepository{record: record}
	reconciler := NewReconciler(repository, gateway, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); err != nil {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if repository.release.State != StateBlocked || !strings.Contains(repository.release.Error, ErrOwnership.Error()) {
		t.Fatalf("release = %#v", repository.release)
	}
}

func TestReconcilerBlocksInvalidAssignment(t *testing.T) {
	record := reconciliationRecord(t)
	repository := &reconcileRepository{record: record}
	reconciler := NewReconciler(repository, &reconcileGateway{assignmentError: ErrInvalidAssignment}, fixedClock{time.Now()}, NewSignal(), "worker", 1, 1)
	if err := reconciler.ReconcileDue(context.Background()); err != nil {
		t.Fatalf("ReconcileDue() error = %v", err)
	}
	if repository.release.State != StateBlocked || !strings.Contains(repository.release.Error, ErrInvalidAssignment.Error()) {
		t.Fatalf("release = %#v", repository.release)
	}
}

func reconciliationRecord(t *testing.T) Record {
	t.Helper()
	definition := validDefinition()
	hash, err := definition.Payload.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	return Record{Definition: definition, ID: "record-id", DiscordMessageID: "remote", DesiredHash: hash, Revision: 1, State: StateRepairing}
}
