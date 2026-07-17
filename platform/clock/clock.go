// Package clock contains deterministic clock and ticker abstractions.
package clock

import "time"

// Ticker emits periodic timestamps.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// Clock provides current time and ticker creation.
type Clock interface {
	Now() time.Time
	NewTicker(time.Duration) Ticker
}

// Real is the system clock implementation.
type Real struct{}

// Now returns the current system time.
func (Real) Now() time.Time {
	return time.Now()
}

// NewTicker creates a system ticker.
func (Real) NewTicker(interval time.Duration) Ticker {
	return realTicker{Ticker: time.NewTicker(interval)}
}

type realTicker struct {
	*time.Ticker
}

func (ticker realTicker) C() <-chan time.Time {
	return ticker.Ticker.C
}
