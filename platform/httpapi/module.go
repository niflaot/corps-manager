package httpapi

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/announcements"
	"github.com/niflaot/corps-manager/internal/inactivity"
	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/internal/performance"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Config contains HTTP API security and request limits.
type Config struct {
	// APIKey authenticates private API routes.
	APIKey string `env:"DISCORD_BOT_API_KEY"`
	// BodyLimit is the maximum accepted request body size.
	BodyLimit int `env:"DISCORD_BOT_HTTP_BODY_LIMIT" envDefault:"1048576"`
}

// Module provides HTTP configuration, routes, Fiber application, and server.
var Module = fx.Module("httpapi", fx.Provide(
	LoadConfig,
	fx.Annotate(provideDependencies, fx.ParamTags("", "", "", "", `name:"customer_page"`)),
	provideApplication, provideServer,
))

// LoadConfig reads and validates HTTP API configuration.
func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	config.APIKey = strings.TrimSpace(config.APIKey)
	if config.APIKey == "" {
		return Config{}, fmt.Errorf("DISCORD_BOT_API_KEY is required")
	}
	if config.BodyLimit <= 0 {
		return Config{}, fmt.Errorf("DISCORD_BOT_HTTP_BODY_LIMIT must be positive")
	}
	return config, nil
}

func provideDependencies(messageService *messages.Service, performanceService *performance.Service,
	inactivityService *inactivity.Service, announcementService *announcements.Service,
	customerPage fiber.Handler) Dependencies {
	return Dependencies{Messages: messageService, Performance: performanceService, Inactivity: inactivityService,
		Announcements: announcementService, CustomerPage: customerPage}
}

func provideApplication(log *zap.Logger, appConfig appconfig.Config, apiConfig Config,
	healthService *health.Service, dependencies Dependencies, version appconfig.Version) *fiber.App {
	return New(log, appConfig, apiConfig, healthService, dependencies, string(version))
}

func provideServer(application *fiber.App, config appconfig.Config, log *zap.Logger,
	version appconfig.Version) *Server {
	return NewServer(application, config, log, string(version))
}
