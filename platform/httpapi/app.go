// Package httpapi contains the Fiber HTTP driving adapter.
package httpapi

import (
	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"go.uber.org/zap"
)

// ErrorResponse is a JSON error response body.
type ErrorResponse struct {
	// Error stores the human-readable error message.
	Error string `json:"error"`
}

// StatusResponse is the public server status response body.
type StatusResponse struct {
	// Status stores the service health label.
	Status string `json:"status"`
	// Environment stores the runtime environment.
	Environment appconfig.Environment `json:"environment"`
	// Version stores the running build version.
	Version string `json:"version"`
	// Dependencies contains current infrastructure availability.
	Dependencies map[string]health.Status `json:"dependencies"`
}

// New creates the Fiber application.
func New(log *zap.Logger, config appconfig.Config, healthService *health.Service, version string) *fiber.App {
	application := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
	})
	application.Use(fiberzap.New(fiberzap.Config{
		Logger:   log,
		Fields:   []string{"latency", "status", "method", "url", "error"},
		Messages: []string{"http server request failed", "http client request failed", "http request completed"},
	}))
	registerRoutes(application, config, healthService, version)
	return application
}

func errorHandler(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if fiberError, ok := err.(*fiber.Error); ok {
		code = fiberError.Code
	}
	return ctx.Status(code).JSON(ErrorResponse{Error: err.Error()})
}
