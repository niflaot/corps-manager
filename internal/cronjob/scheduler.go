// Package cronjob schedules reusable periodic background jobs.
package cronjob

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/zap"
)

// Handler executes one scheduled job iteration.
type Handler func(context.Context) error

// Job describes a periodic background task.
type Job struct {
	// Name identifies the job in logs and diagnostics.
	Name string
	// Interval controls how often the job runs.
	Interval time.Duration
	// Handler contains the job behavior.
	Handler Handler
	// RunOnStart executes the handler once before waiting for the first tick.
	RunOnStart bool
}

// Scheduler runs registered jobs until its context is canceled.
type Scheduler struct {
	clock clock.Clock
	log   *zap.Logger
	jobs  []Job
}

// New creates an empty scheduler.
func New(jobClock clock.Clock, log *zap.Logger) *Scheduler {
	return &Scheduler{clock: jobClock, log: log}
}

// Register validates and adds a periodic job.
func (scheduler *Scheduler) Register(job Job) error {
	if job.Name == "" {
		return fmt.Errorf("cron job name is required")
	}
	if job.Interval <= 0 {
		return fmt.Errorf("cron job %q interval must be positive", job.Name)
	}
	if job.Handler == nil {
		return fmt.Errorf("cron job %q handler is required", job.Name)
	}
	scheduler.jobs = append(scheduler.jobs, job)
	return nil
}

// Run starts every registered job and blocks until cancellation.
func (scheduler *Scheduler) Run(ctx context.Context) error {
	var workers sync.WaitGroup
	workers.Add(len(scheduler.jobs))
	for _, job := range scheduler.jobs {
		go func() {
			defer workers.Done()
			scheduler.run(ctx, job)
		}()
	}
	<-ctx.Done()
	workers.Wait()
	return nil
}

func (scheduler *Scheduler) run(ctx context.Context, job Job) {
	if job.RunOnStart {
		scheduler.execute(ctx, job)
	}
	ticker := scheduler.clock.NewTicker(job.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			scheduler.execute(ctx, job)
		}
	}
}

func (scheduler *Scheduler) execute(ctx context.Context, job Job) {
	if err := job.Handler(ctx); err != nil {
		scheduler.log.Error("cron job failed", zap.String("job", job.Name), zap.Error(err))
	}
}
