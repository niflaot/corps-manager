package settings

import (
	"github.com/pixelados-net/discord-bot/internal/localization"
	"go.uber.org/fx"
)

// Module provides the typed settings service.
var Module = fx.Module("settings", fx.Provide(provideService))

func provideService(repository Repository, catalog *localization.Catalog) *Service {
	return NewService(repository, catalog.Text(localization.VerificationTrapWarningKey))
}
