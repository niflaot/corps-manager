package notification

import (
	"context"

	"go.uber.org/zap"
)

// Signal coalesces immediate outbox delivery wakeups.
type Signal struct {
	// channel contains at most one pending wakeup.
	channel chan struct{}
}

// NewSignal creates a non-blocking delivery signal.
func NewSignal() *Signal { return &Signal{channel: make(chan struct{}, 1)} }

// Notify schedules one immediate outbox sweep.
func (signal *Signal) Notify() {
	select {
	case signal.channel <- struct{}{}:
	default:
	}
}

// C exposes delivery wakeups.
func (signal *Signal) C() <-chan struct{} { return signal.channel }

// Service durably publishes informational verification transitions.
type Service struct {
	// repository persists the durable event.
	repository Repository
	// signal wakes the local dispatcher after insertion.
	signal *Signal
	// log records non-fatal publication failures.
	log *zap.Logger
}

// NewService creates a durable verification notification publisher.
func NewService(repository Repository, signal *Signal, log *zap.Logger) *Service {
	return &Service{repository: repository, signal: signal, log: log}
}

// Enqueue persists an event without making the primary verification operation fail.
func (service *Service) Enqueue(ctx context.Context, event Event) {
	if !event.valid() {
		service.log.Error("invalid verification notification event",
			zap.String("kind", string(event.Kind)), zap.String("user", event.UserID))
		return
	}
	inserted, err := service.repository.Enqueue(ctx, event)
	if err != nil {
		service.log.Error("verification notification enqueue failed",
			zap.String("kind", string(event.Kind)), zap.String("user", event.UserID), zap.Error(err))
		return
	}
	if inserted {
		service.signal.Notify()
	}
}
