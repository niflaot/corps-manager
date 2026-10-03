package bootstrap

import (
	"github.com/niflaot/corps-manager/internal/agreements"
	agreementspostgres "github.com/niflaot/corps-manager/internal/agreements/postgres"
	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/internal/messages"
	messagespostgres "github.com/niflaot/corps-manager/internal/messages/postgres"
	"github.com/niflaot/corps-manager/internal/security"
	"github.com/niflaot/corps-manager/platform/agreementdiscord"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/clock"
	"github.com/niflaot/corps-manager/platform/discord"
	"github.com/niflaot/corps-manager/platform/health"
	"github.com/niflaot/corps-manager/platform/httpapi"
	"github.com/niflaot/corps-manager/platform/logger"
	"github.com/niflaot/corps-manager/platform/postgres"
	"github.com/niflaot/corps-manager/platform/securitydiscord"
	"go.uber.org/fx"
)

// Module composes package-owned modules without declaring domain providers.
var Module = fx.Module("bootstrap",
	appconfig.Module,
	clock.Module,
	logger.Module,
	postgres.Module,
	messagespostgres.Module,
	agreementspostgres.Module,
	discord.Module,
	messages.Module,
	security.Module,
	securitydiscord.Module,
	agreements.Module,
	agreementdiscord.Module,
	health.Module,
	cronjob.Module,
	httpapi.Module,
	fx.Provide(newRuntime),
)
