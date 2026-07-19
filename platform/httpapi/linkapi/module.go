package linkapi

import (
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
	"go.uber.org/fx"
)

// Module provides Discord account-link HTTP configuration and routes.
var Module = fx.Module("discordlinks-http", fx.Provide(LoadConfig, provideRoutes))

func provideRoutes(service *discordlinks.Service, config Config) *Routes {
	return New(service, config)
}
