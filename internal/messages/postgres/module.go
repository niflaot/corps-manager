package postgres

import (
	"github.com/pixelados-net/discord-bot/internal/messages"
	platformpostgres "github.com/pixelados-net/discord-bot/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL managed-message repository.
var Module = fx.Module("messages-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(messages.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
