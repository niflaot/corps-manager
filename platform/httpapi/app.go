// Package httpapi contains the Fiber HTTP driving adapter.
package httpapi

import (
	"context"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/messages"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"go.uber.org/zap"
)

// MessageService contains HTTP-facing managed-message use cases.
type MessageService interface {
	// Create persists one managed message definition.
	Create(context.Context, messages.Definition, string) (messages.MutationResult, error)
	// Get reads one managed message definition.
	Get(context.Context, string) (messages.Record, error)
	// List reads one filtered page.
	List(context.Context, messages.ListQuery) (messages.Page, error)
	// Replace applies one revision-guarded definition update.
	Replace(context.Context, string, uint64, messages.Definition, string) (messages.MutationResult, error)
	// Archive stops managing one definition.
	Archive(context.Context, string, uint64, string) (messages.MutationResult, error)
	// Reconcile schedules one immediate integrity check.
	Reconcile(context.Context, string) error
}

// Dependencies contains optional HTTP use-case dependencies.
type Dependencies struct {
	// Messages manages static Discord message definitions.
	Messages MessageService
}

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
func New(log *zap.Logger, config appconfig.Config, apiConfig Config, healthService *health.Service, dependencies Dependencies, version string) *fiber.App {
	application := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
		BodyLimit:             apiConfig.BodyLimit,
	})
	application.Use(fiberzap.New(fiberzap.Config{
		Logger:   log,
		Fields:   []string{"latency", "status", "method", "url", "error"},
		Messages: []string{"http server request failed", "http client request failed", "http request completed"},
	}))
	registerRoutes(application, config, apiConfig, healthService, dependencies, version)
	return application
}

func errorHandler(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if fiberError, ok := err.(*fiber.Error); ok {
		code = fiberError.Code
	}
	return ctx.Status(code).JSON(ErrorResponse{Error: err.Error()})
}
