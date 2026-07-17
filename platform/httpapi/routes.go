package httpapi

import (
	"github.com/gofiber/fiber/v2"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/health"
	"github.com/pixelados-net/discord-bot/platform/httpapi/openapi"
)

func registerRoutes(application *fiber.App, config appconfig.Config, healthService *health.Service, version string) {
	application.Get("/status", func(ctx *fiber.Ctx) error {
		return ctx.JSON(StatusResponse{
			Status:       "ok",
			Environment:  config.Environment,
			Version:      version,
			Dependencies: healthService.Snapshot(ctx.UserContext()),
		})
	})
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
	application.Use(func(*fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "route not found")
	})
}
