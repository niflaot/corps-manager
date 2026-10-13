package agreements

import (
	"context"
	"errors"

	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/internal/settings"
	"go.uber.org/fx"
)

const refreshJobName = "agreements-panel-refresh"

// Module provides agreement use cases and panel scheduling.
var Module = fx.Module("agreements", fx.Provide(
	LoadConfig,
	fx.Annotate(provideService, fx.ParamTags("", "", "", `name:"guild_id"`, "")),
	fx.Annotate(provideRefreshJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(config Config, repository Repository, messageService *messages.Service, guildID string, runtime settings.Store) *Service {
	return NewService(config, repository, messageService, guildID, runtime)
}

func provideRefreshJob(config Config, service *Service) cronjob.Job {
	return cronjob.Job{Name: refreshJobName, Interval: config.RefreshInterval, RunOnStart: true,
		Handler: func(ctx context.Context) error {
			if _, err := service.Reload(ctx); err != nil {
				return err
			}
			err := service.Publish(ctx)
			if errors.Is(err, ErrDisabled) {
				return nil
			}
			return err
		}}
}
