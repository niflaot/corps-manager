package cronjob

import (
	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the configured application cron scheduler.
var Module = fx.Module("cronjob", fx.Provide(provideScheduler))

type schedulerParams struct {
	fx.In

	// Clock provides scheduler timing.
	Clock clock.Clock
	// Log records job failures.
	Log *zap.Logger
	// Jobs contains package-owned cron job contributions.
	Jobs []Job `group:"cronjobs"`
}

func provideScheduler(params schedulerParams) (*Scheduler, error) {
	scheduler := New(params.Clock, params.Log)
	for _, job := range params.Jobs {
		if err := scheduler.Register(job); err != nil {
			return nil, err
		}
	}
	return scheduler, nil
}
