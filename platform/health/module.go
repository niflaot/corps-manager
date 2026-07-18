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
			log.Info("dependency health", zap.Any("dependencies", service.Snapshot(ctx)))
			return nil
		},
	}
}
