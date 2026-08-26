package cronjob

import (
	"context"
	"testing"
	"time"

	"github.com/niflaot/corps-manager/platform/clock"
	"go.uber.org/zap"
)

func TestSchedulerRunsRegisteredJob(t *testing.T) {
	jobClock := clock.NewFake(time.Now())
	scheduler := New(jobClock, zap.NewNop())
	runs := make(chan struct{}, 1)
	if err := scheduler.Register(Job{Name: "test", Interval: time.Minute, Handler: func(context.Context) error {
		runs <- struct{}{}
		return nil
	}}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	deadline := time.After(time.Second)
	for {
		jobClock.Advance(time.Minute)
		select {
		case <-runs:
			cancel()
			if err := <-done; err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			return
		case <-deadline:
			cancel()
			t.Fatal("job did not run")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestSchedulerRejectsInvalidJob(t *testing.T) {
	scheduler := New(clock.Real{}, zap.NewNop())
	if err := scheduler.Register(Job{}); err == nil {
		t.Fatal("Register() error = nil")
	}
}

func TestSchedulerRunsJobOnStart(t *testing.T) {
	scheduler := New(clock.NewFake(time.Now()), zap.NewNop())
	runs := make(chan struct{}, 1)
	if err := scheduler.Register(Job{Name: "startup", Interval: time.Hour, RunOnStart: true,
		Handler: func(context.Context) error { runs <- struct{}{}; return nil }}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	select {
	case <-runs:
		cancel()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatal("startup job did not run")
	}
}
