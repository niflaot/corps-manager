package security

import (
	"context"
	"time"

	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/internal/messages"
	"go.uber.org/fx"
)

const integrityJobName = "security-integrity"
const integrityTimeout = 45 * time.Second

// Module provides the security policies and their periodic integrity job.
var Module = fx.Module("security", fx.Provide(
	LoadConfig,
	fx.Annotate(provideService, fx.ParamTags("", `name:"guild_id"`, "", "")),
	fx.Annotate(provideJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(config Config, guildID string, store *messages.Service, gateway Gateway) *Service {
	return NewService(config, guildID, store, gateway)
}

func provideJob(config Config, service *Service) cronjob.Job {
	return cronjob.Job{Name: integrityJobName, Interval: config.RefreshInterval, RunOnStart: true, Handler: func(ctx context.Context) error {
		checkContext, cancel := context.WithTimeout(ctx, integrityTimeout)
		defer cancel()
		return service.Refresh(checkContext)
	}}
}
