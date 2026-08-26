package httpapi

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/inactivity"
	"github.com/niflaot/corps-manager/internal/performance"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"github.com/niflaot/corps-manager/platform/httpapi/openapi"
)

func registerRoutes(application *fiber.App, config appconfig.Config, apiConfig Config, healthService *health.Service, dependencies Dependencies, version string) {
	application.Get("/status", func(ctx *fiber.Ctx) error {
		return ctx.JSON(StatusResponse{
			Status:       "ok",
			Environment:  config.Environment,
			Version:      version,
			Dependencies: healthService.Snapshot(ctx.UserContext()),
		})
	})
	if config.Environment.IsDevelopment() {
		registerDocumentationRoutes(application)
	}
	if dependencies.Messages != nil {
		registerMessageRoutes(application.Group("/api/messages", authenticate(apiConfig.APIKey)), dependencies.Messages)
	}
	if dependencies.Performance != nil {
		registerPerformanceRoutes(application.Group("/api/performance", authenticate(apiConfig.APIKey)), dependencies.Performance)
	}
	if dependencies.Inactivity != nil {
		registerInactivityRoutes(application.Group("/api/inactivity", authenticate(apiConfig.APIKey)), dependencies.Inactivity)
	}
	application.Use(func(*fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "route not found")
	})
}

func registerInactivityRoutes(router fiber.Router, service InactivityService) {
	router.Get("/", func(ctx *fiber.Ctx) error {
		entries, err := service.List(ctx.UserContext())
		if err != nil {
			return inactivityError(err)
		}
		return ctx.JSON(fiber.Map{"items": entries, "total": len(entries)})
	})
	router.Post("/", func(ctx *fiber.Ctx) error {
		var request InactivityMutationRequest
		if err := ctx.BodyParser(&request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
		}
		entry, err := service.Add(ctx.UserContext(), request.Name, "api")
		if err != nil {
			return inactivityError(err)
		}
		return ctx.Status(fiber.StatusCreated).JSON(entry)
	})
	router.Delete("/:name", func(ctx *fiber.Ctx) error {
		if err := service.Remove(ctx.UserContext(), ctx.Params("name")); err != nil {
			return inactivityError(err)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})
}

func registerDocumentationRoutes(application *fiber.App) {
	application.Get("/openapi.json", func(ctx *fiber.Ctx) error {
		ctx.Type("json")
		return ctx.SendString(openapi.Spec)
	})
	application.Get("/docs", func(ctx *fiber.Ctx) error {
		ctx.Type("html")
		return ctx.SendString(`<!doctype html>
<html>
<head>
  <title>discord-bot API</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <script id="api-reference" type="application/json">` + openapi.Spec + `</script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`)
	})
}

func registerPerformanceRoutes(router fiber.Router, service PerformanceService) {
	router.Get("/", func(ctx *fiber.Ctx) error {
		state, err := service.Get(ctx.UserContext())
		if err != nil {
			return performanceError(err)
		}
		return ctx.JSON(state)
	})
	router.Post("/refresh", func(ctx *fiber.Ctx) error {
		state, err := service.Refresh(ctx.UserContext())
		if err != nil {
			return performanceError(err)
		}
		return ctx.JSON(state)
	})
}

func performanceError(err error) error {
	switch {
	case errors.Is(err, performance.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, performance.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, performance.ErrDisabled):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	default:
		return fiber.NewError(fiber.StatusBadGateway, "performance refresh failed")
	}
}

func inactivityError(err error) error {
	switch {
	case errors.Is(err, inactivity.ErrInvalidName):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, inactivity.ErrAlreadyExists):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, inactivity.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, inactivity.ErrDisabled):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "inactivity registry operation failed")
	}
}
