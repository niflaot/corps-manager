package notification

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/zap"
)

// Dispatcher delivers claimed verification notifications with retry and DLQ handling.
type Dispatcher struct {
	// repository leases and updates durable deliveries.
	repository Repository
	// sender performs idempotent Discord delivery.
	sender Sender
	// clock supplies deterministic retry timestamps.
	clock clock.Clock
	// signal receives immediate delivery wakeups.
	signal *Signal
	// log records retry and dead-letter transitions.
	log *zap.Logger
	// config controls bounded delivery behavior.
	config Config
	// owner identifies this process in leases.
	owner string
	// runLock prevents overlapping local sweeps.
	runLock sync.Mutex
}

// NewDispatcher creates a bounded durable notification dispatcher.
func NewDispatcher(repository Repository, sender Sender, jobClock clock.Clock, signal *Signal,
	log *zap.Logger, config Config, owner string) *Dispatcher {
	return &Dispatcher{repository: repository, sender: sender, clock: jobClock, signal: signal,
		log: log, config: config, owner: owner}
}

// Run processes immediate notification wakeups until cancellation.
func (dispatcher *Dispatcher) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-dispatcher.signal.C():
			if err := dispatcher.DispatchDue(ctx); err != nil {
				dispatcher.log.Error("verification notification dispatch failed", zap.Error(err))
			}
		}
	}
}

// DispatchDue claims and delivers one bounded notification batch.
func (dispatcher *Dispatcher) DispatchDue(ctx context.Context) error {
	if !dispatcher.runLock.TryLock() {
		return nil
	}
	defer dispatcher.runLock.Unlock()
	now := dispatcher.clock.Now()
	deliveries, err := dispatcher.repository.ClaimDue(ctx, ClaimRequest{
		Owner: dispatcher.owner, Limit: dispatcher.config.BatchSize,
		LeaseDuration: dispatcher.config.LeaseDuration, Now: now,
	})
	if err != nil {
		return err
	}
	jobs := make(chan Delivery)
	failures := make(chan error, len(deliveries))
	var workers sync.WaitGroup
	workerCount := min(dispatcher.config.Workers, len(deliveries))
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for delivery := range jobs {
				if deliveryError := dispatcher.deliver(ctx, delivery); deliveryError != nil {
					failures <- deliveryError
				}
			}
		}()
	}
	for _, delivery := range deliveries {
		select {
		case jobs <- delivery:
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			return ctx.Err()
		}
	}
	close(jobs)
	workers.Wait()
	close(failures)
	var result error
	for failure := range failures {
		result = errors.Join(result, failure)
	}
	return result
}

// deliver completes, retries, or dead-letters one leased notification.
func (dispatcher *Dispatcher) deliver(ctx context.Context, delivery Delivery) error {
	messageID, err := dispatcher.sender.Send(ctx, delivery)
	now := dispatcher.clock.Now()
	if err == nil {
		return dispatcher.repository.Complete(ctx, Completion{
			ID: delivery.ID, Owner: dispatcher.owner, DiscordMessageID: messageID, DeliveredAt: now,
		})
	}
	attempts := delivery.Attempts + 1
	state := StateRetry
	nextAttempt := now.Add(dispatcher.retryDelay(delivery.ID, attempts))
	if attempts >= dispatcher.config.MaxAttempts {
		state, nextAttempt = StateDead, now
	}
	message := err.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	releaseError := dispatcher.repository.Release(ctx, Release{
		ID: delivery.ID, Owner: dispatcher.owner, State: state, Error: message, NextAttemptAt: nextAttempt,
	})
	if state == StateDead {
		dispatcher.log.Error("verification notification moved to dead letter",
			zap.String("notification", delivery.ID), zap.String("kind", string(delivery.Kind)),
			zap.String("user", delivery.UserID), zap.Int("attempts", attempts), zap.Error(err))
	} else {
		dispatcher.log.Warn("verification notification delivery retry scheduled",
			zap.String("notification", delivery.ID), zap.String("kind", string(delivery.Kind)),
			zap.String("user", delivery.UserID), zap.Int("attempts", attempts),
			zap.Time("next_attempt_at", nextAttempt), zap.Error(err))
	}
	return releaseError
}

// retryDelay returns bounded exponential backoff with deterministic jitter.
func (dispatcher *Dispatcher) retryDelay(id string, attempts int) time.Duration {
	exponent := min(max(attempts-1, 0), 20)
	base := dispatcher.config.RetryBase
	for range exponent {
		if base >= dispatcher.config.RetryMax || base > dispatcher.config.RetryMax/2 {
			base = dispatcher.config.RetryMax
			break
		}
		base *= 2
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", id, attempts)))
	jitterLimit := max(dispatcher.config.RetryBase/2, time.Millisecond)
	jitter := time.Duration(int64(digest[0]) * int64(jitterLimit) / 255)
	return min(base+jitter, dispatcher.config.RetryMax)
}
