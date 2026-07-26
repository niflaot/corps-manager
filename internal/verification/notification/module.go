package notification

import (
	"fmt"
	"os"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// dispatchJobName identifies the periodic outbox sweep.
	dispatchJobName = "verification-notifications-dispatch"
)

// Module provides durable verification notification publication and delivery.
var Module = fx.Module("verification-notification", fx.Provide(
	LoadConfig,
	NewSignal,
	fx.Annotate(provideService, fx.As(new(Publisher)), fx.As(fx.Self())),
	provideDispatcher,
	fx.Annotate(provideDispatchJob, fx.ResultTags(`group:"cronjobs"`)),
))

// provideService creates the informational transition publisher.
func provideService(repository Repository, signal *Signal, log *zap.Logger) *Service {
	return NewService(repository, signal, log)
}

// provideDispatcher creates the process-owned delivery worker.
func provideDispatcher(repository Repository, sender Sender, jobClock clock.Clock, signal *Signal,
	log *zap.Logger, config Config) *Dispatcher {
	hostname, _ := os.Hostname()
	owner := fmt.Sprintf("%s:%d:verification-notifications", hostname, os.Getpid())
	return NewDispatcher(repository, sender, jobClock, signal, log, config, owner)
}

// provideDispatchJob contributes the periodic outbox sweep.
func provideDispatchJob(dispatcher *Dispatcher, config Config) cronjob.Job {
	return cronjob.Job{Name: dispatchJobName, Interval: config.Interval, Handler: dispatcher.DispatchDue}
}
