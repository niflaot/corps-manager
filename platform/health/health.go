// Package health runs dependency checks concurrently.
package health

import (
	"context"
	"errors"
	"sync"
)

// ErrDisabled marks an intentionally disabled dependency.
var ErrDisabled = errors.New("dependency is disabled")

// Status is the result of a dependency health check.
type Status string

const (
	// StatusAvailable means the dependency check succeeded.
	StatusAvailable Status = "available"
	// StatusUnavailable means the dependency check failed.
	StatusUnavailable Status = "unavailable"
	// StatusDisabled means the dependency is intentionally disabled.
	StatusDisabled Status = "disabled"
)

// Check verifies one dependency.
type Check func(context.Context) error

// Service executes named dependency checks.
type Service struct {
	checks map[string]Check
}

// New creates a health service with a defensive copy of its checks.
func New(checks map[string]Check) *Service {
	copyChecks := make(map[string]Check, len(checks))
	for name, check := range checks {
		copyChecks[name] = check
	}
	return &Service{checks: copyChecks}
}

// Snapshot executes every check concurrently and returns its status.
func (service *Service) Snapshot(ctx context.Context) map[string]Status {
	statuses := make(map[string]Status, len(service.checks))
	var lock sync.Mutex
	var workers sync.WaitGroup
	workers.Add(len(service.checks))
	for name, check := range service.checks {
		go func() {
			defer workers.Done()
			status := StatusAvailable
			if err := check(ctx); errors.Is(err, ErrDisabled) {
				status = StatusDisabled
			} else if err != nil {
				status = StatusUnavailable
			}
			lock.Lock()
			statuses[name] = status
			lock.Unlock()
		}()
	}
	workers.Wait()
	return statuses
}
