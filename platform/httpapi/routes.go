package httpapi

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/performance"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"github.com/pixelados-net/discord-bot/platform/httpapi/openapi"
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
	application.Use(func(*fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "route not found")
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
