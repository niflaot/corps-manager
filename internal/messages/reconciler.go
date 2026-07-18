package messages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Clock supplies deterministic reconciliation time.
type Clock interface {
	// Now returns the current deterministic time.
	Now() time.Time
}

// Reconciler repairs Discord drift from PostgreSQL desired state.
type Reconciler struct {
	repository Repository
	gateway    Gateway
	clock      Clock
	trigger    Trigger
	owner      string
	batchSize  int
	workers    int
	runLock    sync.Mutex
}

// NewReconciler creates a bounded managed-message reconciler.
func NewReconciler(repository Repository, gateway Gateway, clock Clock, trigger Trigger, owner string, batchSize int, workers int) *Reconciler {
	if batchSize <= 0 {
		batchSize = 25
	}
	if workers <= 0 {
		workers = 4
	}
	return &Reconciler{repository: repository, gateway: gateway, clock: clock, trigger: trigger, owner: owner, batchSize: batchSize, workers: workers}
}

// Run processes immediate API triggers until cancellation.
func (reconciler *Reconciler) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-reconciler.trigger.C():
			_ = reconciler.ReconcileDue(ctx)
		}
	}
}

// ReconcileDue claims and processes one bounded due batch.
func (reconciler *Reconciler) ReconcileDue(ctx context.Context) error {
	if !reconciler.runLock.TryLock() {
		return nil
	}
	defer reconciler.runLock.Unlock()
	now := reconciler.clock.Now()
	records, err := reconciler.repository.ClaimDue(ctx, ClaimRequest{Owner: reconciler.owner, Limit: reconciler.batchSize, LeaseDuration: 2 * time.Minute, Now: now})
	if err != nil {
		return err
	}
	jobs := make(chan Record)
	errorsChannel := make(chan error, len(records))
	var wait sync.WaitGroup
	workerCount := min(reconciler.workers, len(records))
	wait.Add(workerCount)
	for range workerCount {
		go func() {
			defer wait.Done()
			for record := range jobs {
				if err := reconciler.reconcile(ctx, record); err != nil {
					errorsChannel <- err
				}
			}
		}()
	}
	for _, record := range records {
		select {
		case jobs <- record:
		case <-ctx.Done():
			close(jobs)
			wait.Wait()
			return ctx.Err()
		}
	}
	close(jobs)
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		return err
	}
	return nil
}

func (reconciler *Reconciler) reconcile(ctx context.Context, record Record) error {
	now := reconciler.clock.Now()
	if err := reconciler.gateway.ValidateAssignment(ctx, record.GuildID, record.ChannelID); err != nil {
		return reconciler.release(ctx, record, now, err)
	}
	observed, repaired, err := reconciler.ensure(ctx, record)
	if err != nil {
		return reconciler.release(ctx, record, now, err)
	}
	hash, err := observed.Payload.Hash()
	if err != nil || hash != record.DesiredHash {
		if err == nil {
			err = fmt.Errorf("discord response hash does not match desired state")
		}
		return reconciler.release(ctx, record, now, err)
	}
	return reconciler.repository.Complete(ctx, Completion{ID: record.ID, Owner: reconciler.owner, Revision: record.Revision, DiscordMessageID: observed.ID, ObservedHash: hash, Repaired: repaired, CheckedAt: now, NextCheckAt: now.Add(5 * time.Minute)})
}

func (reconciler *Reconciler) ensure(ctx context.Context, record Record) (ObservedMessage, bool, error) {
	if record.DiscordMessageID == "" {
		observed, err := reconciler.create(ctx, record)
		return observed, true, err
	}
	observed, err := reconciler.gateway.Get(ctx, record.ChannelID, record.DiscordMessageID)
	if errors.Is(err, ErrNotFound) {
		created, createErr := reconciler.create(ctx, record)
		return created, true, createErr
	}
	if err != nil {
		return ObservedMessage{}, false, err
	}
	if !observed.Owned {
		return ObservedMessage{}, false, ErrOwnership
	}
	hash, err := observed.Payload.Hash()
	if err != nil {
		return ObservedMessage{}, false, err
	}
	if hash == record.DesiredHash {
		return observed, false, nil
	}
	replaced, err := reconciler.gateway.Replace(ctx, ReplaceRequest{ChannelID: record.ChannelID, MessageID: record.DiscordMessageID, Payload: record.Payload})
	if err != nil {
		return ObservedMessage{}, false, err
	}
	if !replaced.Owned {
		return ObservedMessage{}, false, ErrOwnership
	}
	return replaced, true, nil
}

func (reconciler *Reconciler) create(ctx context.Context, record Record) (ObservedMessage, error) {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", record.ID, record.Revision, record.ChannelID)))
	nonce := hex.EncodeToString(digest[:])[:25]
	created, err := reconciler.gateway.Create(ctx, CreateRequest{ChannelID: record.ChannelID, Payload: record.Payload, Nonce: nonce})
	if err == nil && !created.Owned {
		return ObservedMessage{}, ErrOwnership
	}
	return created, err
}

func (reconciler *Reconciler) release(ctx context.Context, record Record, now time.Time, err error) error {
	state := StateDrifted
	delay := retryDelay(record.ID, record.FailureCount)
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrOwnership) || errors.Is(err, ErrInvalidRemote) || errors.Is(err, ErrInvalidAssignment) || errors.Is(err, ErrAmbiguousCreate) {
		state = StateBlocked
		delay = 30 * time.Minute
	}
	message := err.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	return reconciler.repository.Release(ctx, Release{ID: record.ID, Owner: reconciler.owner, Revision: record.Revision, State: state, Error: message, CheckedAt: now, NextCheckAt: now.Add(delay)})
}

func retryDelay(id string, failures int) time.Duration {
	failures = min(max(failures, 0), 5)
	base := time.Minute * time.Duration(1<<failures)
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", id, failures)))
	return min(base+time.Duration(digest[0]%30)*time.Second, 30*time.Minute)
}
