package bootstrap

import (
	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/internal/inactivity"
	inactivitypostgres "github.com/niflaot/corps-manager/internal/inactivity/postgres"
	"github.com/niflaot/corps-manager/internal/messages"
	messagespostgres "github.com/niflaot/corps-manager/internal/messages/postgres"
	"github.com/niflaot/corps-manager/internal/performance"
	performancepostgres "github.com/niflaot/corps-manager/internal/performance/postgres"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/clock"
	"github.com/niflaot/corps-manager/platform/discord"
	"github.com/niflaot/corps-manager/platform/health"
	"github.com/niflaot/corps-manager/platform/httpapi"
	"github.com/niflaot/corps-manager/platform/logger"
	"github.com/niflaot/corps-manager/platform/postgres"
	"github.com/niflaot/corps-manager/platform/sarp"
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
	inactivitypostgres.Module,
	discord.Module,
	sarp.Module,
	messages.Module,
	performance.Module,
	inactivity.Module,
	health.Module,
	cronjob.Module,
	httpapi.Module,
	fx.Provide(newRuntime),
)
