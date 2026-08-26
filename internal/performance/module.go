package performance

import (
	"context"
	"errors"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const refreshJobName = "business-performance-refresh"

// Module provides business performance collection and scheduling.
var Module = fx.Module("performance", fx.Provide(
	LoadConfig,
	fx.Annotate(provideService, fx.ParamTags("", "", "", "", "", "", `name:"guild_id"`)),
	fx.Annotate(provideRefreshJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(config Config, source Source, repository Repository, messageService *messages.Service,
	serviceClock clock.Clock, log *zap.Logger, guildID string) *Service {
	return NewService(config, source, repository, messageService, serviceClock, log, guildID)
}

func provideRefreshJob(config Config, service *Service, log *zap.Logger) cronjob.Job {
	return cronjob.Job{Name: refreshJobName, Interval: config.Interval, RunOnStart: config.Enabled,
		Handler: func(ctx context.Context) error {
			_, err := service.Refresh(ctx)
			if errors.Is(err, ErrDisabled) {
				log.Debug("business performance collection disabled")
				return nil
			}
			return err
		}}
}
