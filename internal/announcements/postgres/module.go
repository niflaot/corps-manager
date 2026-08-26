// Package postgres persists announcement cooldowns in PostgreSQL.
package postgres

import (
	"github.com/niflaot/corps-manager/internal/announcements"
	platformpostgres "github.com/niflaot/corps-manager/platform/postgres"
	"go.uber.org/fx"
)

// Module provides the PostgreSQL announcement repository.
var Module = fx.Module("announcements-postgres", fx.Provide(
	fx.Annotate(provideStore, fx.As(new(announcements.Repository))),
))

func provideStore(pool *platformpostgres.Pool) *Store { return NewStore(pool.DB()) }
