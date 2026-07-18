// Package bootstrap wires and runs the application infrastructure.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/internal/messages"
	messagespostgres "github.com/pixelados-net/discord-bot/internal/messages/postgres"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"github.com/pixelados-net/discord-bot/platform/discord"
	"github.com/pixelados-net/discord-bot/platform/health"
	"github.com/pixelados-net/discord-bot/platform/httpapi"
	"github.com/pixelados-net/discord-bot/platform/logger"
	"github.com/pixelados-net/discord-bot/platform/postgres"
	redisplatform "github.com/pixelados-net/discord-bot/platform/redis"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Application owns the fully wired process and its infrastructure.
type Application struct {
	discord    *discord.Client
	postgres   *postgres.Pool
	redis      *redisplatform.Client
	scheduler  *cronjob.Scheduler
	reconciler *messages.Reconciler
	server     *httpapi.Server
	log        *zap.Logger
	closeOnce  sync.Once
}

// New loads configuration and wires every runtime dependency.
func New(ctx context.Context, version string) (*Application, error) {
	config, err := appconfig.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load app config: %w", err)
	}
	discordConfig, err := discord.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load discord config: %w", err)
	}
	apiConfig, err := httpapi.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load http config: %w", err)
	}
	log, err := logger.New(config.Environment.IsDevelopment())
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	discordClient, err := discord.New(discordConfig, log)
	if err != nil {
		return nil, err
	}
	redisConfig, err := redisplatform.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load redis config: %w", err)
	}
	redisClient := redisplatform.New(redisConfig)
	postgresConfig, err := postgres.LoadConfig()
	if err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("load postgres config: %w", err)
	}
	postgresPool, err := postgres.New(ctx, postgresConfig)
	if err != nil {
		_ = redisClient.Close()
		return nil, err
	}
	messageStore := messagespostgres.NewStore(postgresPool.DB())
	messageGateway := discord.NewMessageGateway(discordClient)
	messageSignal := messages.NewSignal()
	messageService := messages.NewService(messageStore, messageSignal)
	hostname, _ := os.Hostname()
	messageReconciler := messages.NewReconciler(messageStore, messageGateway, clock.Real{}, messageSignal,
		fmt.Sprintf("%s:%d", hostname, os.Getpid()), 25, 4)
	healthService := newHealthService(redisClient, postgresPool, discordClient)
	scheduler := cronjob.New(clock.Real{}, log)
	if err := scheduler.Register(healthJob(healthService, log)); err != nil {
		postgresPool.Close()
		_ = redisClient.Close()
		return nil, err
	}
	if err := scheduler.Register(cronjob.Job{Name: "messages-reconcile", Interval: time.Minute, Handler: messageReconciler.ReconcileDue}); err != nil {
		postgresPool.Close()
		_ = redisClient.Close()
		return nil, err
	}
	application := httpapi.New(log, config, apiConfig, healthService, httpapi.Dependencies{Messages: messageService}, version)
	return &Application{
		discord:    discordClient,
		postgres:   postgresPool,
		redis:      redisClient,
		scheduler:  scheduler,
		reconciler: messageReconciler,
		server:     httpapi.NewServer(application, config, log, version),
		log:        log,
	}, nil
}

// Run starts the HTTP server, Discord gateway, and cron scheduler concurrently.
func (application *Application) Run(ctx context.Context) error {
	group, groupContext := errgroup.WithContext(ctx)
	group.Go(func() error { return application.server.Run(groupContext) })
	group.Go(func() error { return application.discord.Run(groupContext) })
	group.Go(func() error { return application.scheduler.Run(groupContext) })
	group.Go(func() error { return application.reconciler.Run(groupContext) })
	return group.Wait()
}

// Close releases the database, cache, and logger resources once.
func (application *Application) Close() {
	application.closeOnce.Do(func() {
		application.postgres.Close()
		if err := application.redis.Close(); err != nil {
			application.log.Warn("close redis", zap.Error(err))
		}
		_ = application.log.Sync()
	})
}

func newHealthService(redisClient *redisplatform.Client, postgresPool *postgres.Pool, discordClient *discord.Client) *health.Service {
	return health.New(map[string]health.Check{
		"redis":    redisClient.Ping,
		"postgres": postgresPool.Ping,
		"discord": func(context.Context) error {
			if !discordClient.Connected() {
				return errors.New("discord gateway is disconnected")
			}
			return nil
		},
	})
}

func healthJob(healthService *health.Service, log *zap.Logger) cronjob.Job {
	return cronjob.Job{
		Name:     "dependency-health",
		Interval: time.Minute,
		Handler: func(ctx context.Context) error {
			log.Info("dependency health", zap.Any("dependencies", healthService.Snapshot(ctx)))
			return nil
		},
	}
}
