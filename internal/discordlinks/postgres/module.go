// Package postgres persists Discord account links in PostgreSQL.
package postgres

import (
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
	platformpostgres "github.com/pixelados-net/discord-bot/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL Discord link repository.
var Module = fx.Module("discordlinks-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(discordlinks.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
