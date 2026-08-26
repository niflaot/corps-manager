package inactivity

import (
	"context"
	"errors"

	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/internal/messages"
	"go.uber.org/fx"
)

const refreshJobName = "inactivity-dashboard-refresh"

// Module provides inactivity registry use cases and dashboard scheduling.
var Module = fx.Module("inactivity", fx.Provide(
	LoadConfig,
	fx.Annotate(provideService, fx.ParamTags("", "", "", `name:"guild_id"`)),
	fx.Annotate(provideRefreshJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(config Config, repository Repository, messageService *messages.Service, guildID string) *Service {
	return NewService(config, repository, messageService, guildID)
}

func provideRefreshJob(config Config, service *Service) cronjob.Job {
	return cronjob.Job{Name: refreshJobName, Interval: config.RefreshInterval, RunOnStart: config.Enabled,
		Handler: func(ctx context.Context) error {
			err := service.Publish(ctx)
			if errors.Is(err, ErrDisabled) {
				return nil
			}
			return err
		}}
}
