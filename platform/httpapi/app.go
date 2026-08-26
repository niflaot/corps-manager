// Package httpapi contains the Fiber HTTP driving adapter.
package httpapi

import (
	"context"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/announcements"
	"github.com/niflaot/corps-manager/internal/inactivity"
	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/internal/performance"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
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

// PerformanceService contains HTTP-facing business performance use cases.
type PerformanceService interface {
	// Get returns the current persisted aggregate.
	Get(context.Context) (performance.State, error)
	// Refresh immediately collects and publishes a new snapshot.
	Refresh(context.Context) (performance.State, error)
}

// InactivityService contains HTTP-facing inactivity registry use cases.
type InactivityService interface {
	// List returns all inactivity dismissal entries.
	List(context.Context) ([]inactivity.Entry, error)
	// Add registers one employee dismissal.
	Add(context.Context, string, string) (inactivity.Entry, error)
	// Remove deletes one employee dismissal.
	Remove(context.Context, string) error
}

// AnnouncementService contains HTTP-facing opening-announcement use cases.
type AnnouncementService interface {
	// AnnounceOpening publishes an opening if its cooldown is available.
	AnnounceOpening(context.Context, string) (announcements.State, error)
	// GetCooldown returns the persisted opening cooldown.
	GetCooldown(context.Context) (announcements.State, error)
	// ClearCooldown makes the opening announcement immediately available.
	ClearCooldown(context.Context) error
}

// Dependencies contains optional HTTP use-case dependencies.
type Dependencies struct {
	// Messages manages static Discord message definitions.
	Messages MessageService
	// Performance manages business earnings collection.
	Performance PerformanceService
	// Inactivity manages employees dismissed for inactivity.
	Inactivity InactivityService
	// Announcements manages public business-opening announcements.
	Announcements AnnouncementService
}

// InactivityMutationRequest contains one Nombre_Apellido registry value.
type InactivityMutationRequest struct {
	// Name is the employee roleplay name.
	Name string `json:"name"`
}

// OpeningAnnouncementRequest contains optional announcement attribution.
type OpeningAnnouncementRequest struct {
	// Actor is shown in the Discord embed footer.
	Actor string `json:"actor"`
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
