package postgres

import (
	"github.com/pixelados-net/discord-bot/internal/verification/notification"
	platformpostgres "github.com/pixelados-net/discord-bot/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL verification notification repository.
var Module = fx.Module("verification-notification-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(notification.Repository))),
))

// provideStore creates the PostgreSQL notification outbox adapter.
func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
