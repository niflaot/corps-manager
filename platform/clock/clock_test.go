package clock

import (
	"testing"
	"time"
)

func TestFakeAdvance(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFake(start)
	ticker := clock.NewTicker(time.Minute)
	clock.Advance(time.Minute)
	if got := <-ticker.C(); !got.Equal(start.Add(time.Minute)) {
		t.Fatalf("tick = %v", got)
	}
	ticker.Stop()
}
