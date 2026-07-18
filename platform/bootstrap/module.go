package bootstrap

import (
	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/internal/localization"
	"github.com/pixelados-net/discord-bot/internal/messages"
	messagespostgres "github.com/pixelados-net/discord-bot/internal/messages/postgres"
	"github.com/pixelados-net/discord-bot/internal/settings"
	settingspostgres "github.com/pixelados-net/discord-bot/internal/settings/postgres"
	"github.com/pixelados-net/discord-bot/internal/verification"
	verificationpostgres "github.com/pixelados-net/discord-bot/internal/verification/postgres"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"github.com/pixelados-net/discord-bot/platform/discord"
	"github.com/pixelados-net/discord-bot/platform/events"
	"github.com/pixelados-net/discord-bot/platform/health"
	"github.com/pixelados-net/discord-bot/platform/httpapi"
	"github.com/pixelados-net/discord-bot/platform/logger"
	"github.com/pixelados-net/discord-bot/platform/postgres"
	redisplatform "github.com/pixelados-net/discord-bot/platform/redis"
	"go.uber.org/fx"
)

// Module composes package-owned modules without declaring domain providers.
var Module = fx.Module("bootstrap",
	appconfig.Module,
	clock.Module,
	logger.Module,
	postgres.Module,
	redisplatform.Module,
	events.Module,
	localization.Module,
	messagespostgres.Module,
	settingspostgres.Module,
	verificationpostgres.Module,
	discord.Module,
	messages.Module,
	settings.Module,
	verification.Module,
	health.Module,
	cronjob.Module,
	httpapi.Module,
	fx.Provide(newRuntime),
)
