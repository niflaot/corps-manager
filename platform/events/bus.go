// Package events provides the process-local application event bus.
package events

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	localbus "github.com/mariuswilms/bus"
	"go.uber.org/zap"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9_-]*)*$`)

var (
	// ErrInvalidName reports an invalid event name.
	ErrInvalidName = errors.New("invalid event name")
	// ErrUnavailable reports a stopped or saturated local event bus.
	ErrUnavailable = errors.New("event bus unavailable")
)

// Event is one process-local event delivery.
type Event struct {
	// ID identifies this publication within the process.
	ID uint64
	// Name is the dot-separated event name.
	Name string
	// Payload contains caller-owned event data.
	Payload any
}

// Handler processes one event sequentially for its subscription.
type Handler func(context.Context, Event) error

// Unsubscribe removes one event subscription.
type Unsubscribe func()

// Bus publishes best-effort events within one process.
type Bus struct {
	broker    *localbus.Broker
	ctx       context.Context
	cancel    context.CancelFunc
	log       *zap.Logger
	closeOnce sync.Once
}

// New creates a context-bound local event bus.
func New(parent context.Context, log *zap.Logger) *Bus {
	ctx, cancel := context.WithCancel(parent)
	return &Bus{broker: localbus.NewBroker(ctx), ctx: ctx, cancel: cancel, log: log}
}

// Publish queues one event without blocking the caller.
func (bus *Bus) Publish(name string, payload any) (uint64, error) {
	if err := validateName(name); err != nil {
		return 0, err
	}
	select {
	case <-bus.ctx.Done():
		return 0, fmt.Errorf("%w: %v", ErrUnavailable, bus.ctx.Err())
	default:
	}
	accepted, id := bus.broker.Publish(name, payload)
	if !accepted {
		return id, ErrUnavailable
	}
	return id, nil
}

// Subscribe registers an asynchronous exact-name handler.
func (bus *Bus) Subscribe(ctx context.Context, name string, handler Handler) (Unsubscribe, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, fmt.Errorf("event handler is required")
	}
	select {
	case <-bus.ctx.Done():
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, bus.ctx.Err())
	default:
	}
	ctx, cancel := context.WithCancel(ctx)
	messages, remove := bus.broker.Subscribe("^" + regexp.QuoteMeta(name) + "$")
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			cancel()
			remove()
		})
	}
	go bus.consume(ctx, messages, handler, unsubscribe)
	return unsubscribe, nil
}

// Close rejects future publications and removes every subscription.
func (bus *Bus) Close() {
	bus.closeOnce.Do(bus.cancel)
}

func (bus *Bus) consume(ctx context.Context, messages <-chan localbus.Message, handler Handler, unsubscribe Unsubscribe) {
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case message, open := <-messages:
			if !open {
				return
			}
			event := Event{ID: message.Id, Name: message.Topic, Payload: message.Data}
			if err := handler(ctx, event); err != nil {
				bus.log.Error("event handler failed", zap.String("event", event.Name), zap.Uint64("event_id", event.ID), zap.Error(err))
			}
		}
	}
}

func validateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("%w: use lowercase dot-separated segments", ErrInvalidName)
	}
	return nil
}
