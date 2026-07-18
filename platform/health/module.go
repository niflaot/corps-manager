package health

import (
	"context"
	"errors"
	"time"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/platform/discord"
	"github.com/pixelados-net/discord-bot/platform/postgres"
	redisplatform "github.com/pixelados-net/discord-bot/platform/redis"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	redisDependencyName         = "redis"
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

func provideService(redisClient *redisplatform.Client, postgresPool *postgres.Pool,
	discordClient *discord.Client) *Service {
	return New(map[string]Check{
		redisDependencyName:    redisClient.Ping,
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
