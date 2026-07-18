// Package httpapi contains the Fiber HTTP driving adapter.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/internal/settings"
	"github.com/pixelados-net/discord-bot/internal/verification"
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

// SettingService contains HTTP-facing setting use cases.
type SettingService interface {
	// Get returns one effective setting.
	Get(context.Context, settings.Key) (settings.Record, error)
	// List returns all effective settings.
	List(context.Context) ([]settings.Record, error)
	// Set persists one typed setting.
	Set(context.Context, settings.Key, json.RawMessage, uint64) (settings.Record, error)
	// Reset deletes one setting override.
	Reset(context.Context, settings.Key, uint64) (settings.Record, error)
}

// VerificationService contains HTTP-facing verification use cases.
type VerificationService interface {
	// CreateGroup creates one verification group.
	CreateGroup(context.Context, verification.Group) (verification.Group, error)
	// UpdateGroup replaces one verification group.
	UpdateGroup(context.Context, string, uint64, verification.Group) (verification.Group, error)
	// GetGroup returns one verification group.
	GetGroup(context.Context, string) (verification.Group, error)
	// ListGroups returns verification groups.
	ListGroups(context.Context, bool) ([]verification.Group, error)
	// DeleteGroup removes one verification group.
	DeleteGroup(context.Context, string, uint64) error
	// ListMemberships returns active memberships.
	ListMemberships(context.Context, string) (verification.Page, error)
	// ReconcileMemberships repairs persisted membership and Discord role state.
	ReconcileMemberships(context.Context) error
}

// VerificationGuard contains the manual guard reconciliation use case.
type VerificationGuard interface {
	// Reconcile repairs verification guard state.
	Reconcile(context.Context) error
}

// GuildService contains HTTP-facing Discord guild statistics use cases.
type GuildService interface {
	// MemberCount returns the guild's approximate member and online presence counts.
	MemberCount(context.Context) (int, int, error)
}

// Dependencies contains optional HTTP use-case dependencies.
type Dependencies struct {
	// Messages manages static Discord message definitions.
	Messages MessageService
	// Settings manages SQL-backed application settings.
	Settings SettingService
	// Verification manages groups and membership inspection.
	Verification VerificationService
	// VerificationGuard repairs Discord verification guard state.
	VerificationGuard VerificationGuard
	// Guild reports public Discord guild statistics.
	Guild GuildService
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

// GuildStats is the public guild population response body.
type GuildStats struct {
	// MemberCount is the guild's approximate total member count.
	MemberCount int `json:"memberCount"`
	// PresenceCount is the guild's approximate online member count.
	PresenceCount int `json:"presenceCount"`
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

func registerSettingRoutes(router fiber.Router, service SettingService) {
	router.Get("/", listSettings(service))
	router.Get("/:key", getSetting(service))
	router.Put("/:key", setSetting(service))
	router.Delete("/:key", resetSetting(service))
}

func listSettings(service SettingService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		records, err := service.List(ctx.UserContext())
		if err != nil {
			return settingError(err)
		}
		return ctx.JSON(fiber.Map{"items": records})
	}
}

func getSetting(service SettingService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		record, err := service.Get(ctx.UserContext(), settings.Key(ctx.Params("key")))
		if err != nil {
			return settingError(err)
		}
		if record.Revision > 0 {
			ctx.Set(fiber.HeaderETag, strconv.FormatUint(record.Revision, 10))
		}
		return ctx.JSON(record)
	}
}

func setSetting(service SettingService) fiber.Handler {
	type request struct {
		Value json.RawMessage `json:"value"`
	}
	return func(ctx *fiber.Ctx) error {
		var body request
		if err := decodeStrict(ctx.Body(), &body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		revision := uint64(0)
		if raw := ctx.Get(fiber.HeaderIfMatch); raw != "" {
			parsed, err := parseRevision(raw)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			revision = parsed
		}
		record, err := service.Set(ctx.UserContext(), settings.Key(ctx.Params("key")), body.Value, revision)
		if err != nil {
			return settingError(err)
		}
		return ctx.JSON(record)
	}
}

func resetSetting(service SettingService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		revision := uint64(0)
		if raw := ctx.Get(fiber.HeaderIfMatch); raw != "" {
			parsed, err := parseRevision(raw)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			revision = parsed
		}
		record, err := service.Reset(ctx.UserContext(), settings.Key(ctx.Params("key")), revision)
		if err != nil {
			return settingError(err)
		}
		return ctx.JSON(record)
	}
}

func settingError(err error) error {
	switch {
	case errors.Is(err, settings.ErrInvalid):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, settings.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, settings.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal server error")
	}
}
