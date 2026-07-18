package postgres

import (
	"github.com/pixelados-net/discord-bot/internal/verification"
	platformpostgres "github.com/pixelados-net/discord-bot/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL verification repository.
var Module = fx.Module("verification-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(verification.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
