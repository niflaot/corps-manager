package discordlinks

import (
	"time"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/fx"
)

const cleanupJobInterval = time.Hour
const cleanupJobName = "discordlinks-cleanup"

// Module provides Discord account-link domain configuration and behavior.
var Module = fx.Module("discordlinks", fx.Provide(
	LoadConfig,
	provideService,
	fx.Annotate(provideCleanupJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(repository Repository, oauth OAuthGateway, serviceClock clock.Clock, config Config) *Service {
	return NewService(repository, oauth, serviceClock, config)
}

func provideCleanupJob(service *Service) cronjob.Job {
	return cronjob.Job{Name: cleanupJobName, Interval: cleanupJobInterval, Handler: service.Cleanup}
}
