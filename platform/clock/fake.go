package clock

import (
	"sync"
	"time"
)

// Fake is a manually advanced clock for deterministic tests.
type Fake struct {
	mu      sync.Mutex
	now     time.Time
	tickers []*FakeTicker
}

// FakeTicker is a manually advanced ticker.
type FakeTicker struct {
	channel chan time.Time
	stopped bool
	mu      sync.Mutex
}

// NewFake creates a fake clock at the supplied time.
func NewFake(now time.Time) *Fake {
	return &Fake{now: now}
}

// Now returns the fake current time.
func (clock *Fake) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

// NewTicker creates a ticker controlled by Advance.
func (clock *Fake) NewTicker(time.Duration) Ticker {
	ticker := &FakeTicker{channel: make(chan time.Time, 1)}
	clock.mu.Lock()
	clock.tickers = append(clock.tickers, ticker)
	clock.mu.Unlock()
	return ticker
}

// Advance moves time forward and ticks every active ticker once.
func (clock *Fake) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	now := clock.now
	tickers := append([]*FakeTicker(nil), clock.tickers...)
	clock.mu.Unlock()
	for _, ticker := range tickers {
		ticker.tick(now)
	}
}

// C returns the ticker channel.
func (ticker *FakeTicker) C() <-chan time.Time {
	return ticker.channel
}

// Stop prevents subsequent ticks.
func (ticker *FakeTicker) Stop() {
	ticker.mu.Lock()
	ticker.stopped = true
	ticker.mu.Unlock()
}

func (ticker *FakeTicker) tick(now time.Time) {
	ticker.mu.Lock()
	defer ticker.mu.Unlock()
	if ticker.stopped {
		return
	}
	select {
	case ticker.channel <- now:
	default:
	}
}
