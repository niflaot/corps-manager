package health

import (
	"context"
	"errors"
	"time"

	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/platform/discord"
	"github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	postgresDependencyName      = "postgres"
	discordDependencyName       = "discord"
	dependencyHealthJobName     = "dependency-health"
	dependencyHealthJobInterval = time.Minute
)

// Module provides aggregate infrastructure health checks.
var Module = fx.Module("health", fx.Provide(
	provideService,
	fx.Annotate(provideHealthJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(postgresPool *postgres.Pool, discordClient *discord.Client) *Service {
	return New(map[string]Check{
		postgresDependencyName: postgresPool.Ping,
		discordDependencyName: func(context.Context) error {
			if !discordClient.Connected() {
				return errors.New("discord gateway is disconnected")
			}
			return nil
		},
	})
}

func provideHealthJob(service *Service, log *zap.Logger) cronjob.Job {
	return cronjob.Job{
		Name:     dependencyHealthJobName,
		Interval: dependencyHealthJobInterval,
		Handler: func(ctx context.Context) error {
			statuses := service.Snapshot(ctx)
			if dependenciesAvailable(statuses) {
				log.Debug("dependency health", zap.Any("dependencies", statuses))
			} else {
				log.Error("dependency health check failed", zap.Any("dependencies", statuses))
			}
			return nil
		},
	}
}

func dependenciesAvailable(statuses map[string]Status) bool {
	for _, status := range statuses {
		if status != StatusAvailable {
			return false
		}
	}
	return true
}
