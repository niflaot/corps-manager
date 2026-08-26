package bootstrap

import (
	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/internal/messages"
	messagespostgres "github.com/pixelados-net/discord-bot/internal/messages/postgres"
	"github.com/pixelados-net/discord-bot/internal/performance"
	performancepostgres "github.com/pixelados-net/discord-bot/internal/performance/postgres"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"github.com/pixelados-net/discord-bot/platform/discord"
	"github.com/pixelados-net/discord-bot/platform/health"
	"github.com/pixelados-net/discord-bot/platform/httpapi"
	"github.com/pixelados-net/discord-bot/platform/logger"
	"github.com/pixelados-net/discord-bot/platform/postgres"
	"github.com/pixelados-net/discord-bot/platform/sarp"
	"go.uber.org/fx"
)

// Module composes package-owned modules without declaring domain providers.
var Module = fx.Module("bootstrap",
	appconfig.Module,
	clock.Module,
	logger.Module,
	postgres.Module,
	messagespostgres.Module,
	performancepostgres.Module,
	discord.Module,
	sarp.Module,
	messages.Module,
	performance.Module,
	health.Module,
	cronjob.Module,
	httpapi.Module,
	fx.Provide(newRuntime),
)
