package verification

import (
	"context"
	"time"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/internal/localization"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/internal/settings"
	"github.com/pixelados-net/discord-bot/internal/verification/notification"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	guardReconcileJobName          = "verification-guard-reconcile"
	guardReconcileJobInterval      = time.Minute
	membershipReconcileJobName     = "verification-memberships-reconcile"
	membershipReconcileJobInterval = time.Minute
)

// Module provides verification services, guard reconciliation, and startup repair.
var Module = fx.Module("verification",
	fx.Provide(
		fx.Annotate(provideService, fx.ParamTags("", "", "", `name:"guild_id"`)),
		provideGuard,
		fx.Annotate(provideGuardReconcileJob, fx.ResultTags(`group:"cronjobs"`)),
		fx.Annotate(provideMembershipReconcileJob, fx.ResultTags(`group:"cronjobs"`)),
	),
	fx.Invoke(registerInitialReconciliation),
)

func provideService(repository Repository, gateway Gateway, notifications notification.Publisher, guildID string) *Service {
	return NewService(repository, gateway, notifications, guildID)
}

func provideGuard(repository Repository, messageService *messages.Service, settingService *settings.Service,
	gateway GuardGateway, catalog *localization.Catalog) *Guard {
	return NewGuard(repository, messageService, settingService, gateway,
		catalog.Text(localization.VerificationUnavailableKey))
}

func provideGuardReconcileJob(guard *Guard) cronjob.Job {
	return cronjob.Job{
		Name:     guardReconcileJobName,
		Interval: guardReconcileJobInterval,
		Handler:  guard.Reconcile,
	}
}

func provideMembershipReconcileJob(service *Service) cronjob.Job {
	return cronjob.Job{
		Name:     membershipReconcileJobName,
		Interval: membershipReconcileJobInterval,
		Handler:  service.ReconcileMemberships,
	}
}

func registerInitialReconciliation(lifecycle fx.Lifecycle, guard *Guard, service *Service, log *zap.Logger) {
	lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		if err := guard.Reconcile(ctx); err != nil {
			log.Warn("initial verification guard reconcile", zap.Error(err))
		}
		if err := service.ReconcileMemberships(ctx); err != nil {
			log.Warn("initial verification memberships reconcile", zap.Error(err))
		}
		return nil
	}})
}
