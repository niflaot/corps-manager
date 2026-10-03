package httpapi

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/agreements"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"github.com/niflaot/corps-manager/platform/httpapi/openapi"
)

func registerRoutes(application *fiber.App, config appconfig.Config, apiConfig Config, healthService *health.Service, dependencies Dependencies, version string) {
	application.Get("/status", func(ctx *fiber.Ctx) error {
		return ctx.JSON(StatusResponse{Status: "ok", Environment: config.Environment, Version: version, Dependencies: healthService.Snapshot(ctx.UserContext())})
	})
	if config.Environment.IsDevelopment() {
		registerDocumentationRoutes(application)
	}
	if dependencies.Messages != nil {
		registerMessageRoutes(application.Group("/api/messages", authenticate(apiConfig.APIKey)), dependencies.Messages)
	}
	if dependencies.Companies != nil {
		registerCompanyRoutes(application.Group("/api/companies", authenticate(apiConfig.APIKey)), dependencies.Companies)
	}
	application.Use(func(*fiber.Ctx) error { return fiber.NewError(fiber.StatusNotFound, "route not found") })
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

func registerCompanyRoutes(router fiber.Router, service CompanyService) {
	router.Get("/", func(ctx *fiber.Ctx) error {
		items, err := service.ListCompanies(ctx.UserContext())
		if err != nil {
			return companyError(err)
		}
		return ctx.JSON(fiber.Map{"items": items, "total": len(items)})
	})
	router.Post("/", func(ctx *fiber.Ctx) error {
		var request agreements.Company
		if err := decodeStrict(ctx.Body(), &request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		company, err := service.CreateCompany(ctx.UserContext(), request.ID, request.Name)
		if err != nil {
			return companyError(err)
		}
		return ctx.Status(fiber.StatusCreated).JSON(company)
	})
	router.Delete("/:id", func(ctx *fiber.Ctx) error {
		if err := service.DeleteCompany(ctx.UserContext(), ctx.Params("id")); err != nil {
			return companyError(err)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	})
}

func companyError(err error) error {
	switch {
	case errors.Is(err, agreements.ErrInvalidCompany):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, agreements.ErrCompanyNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, agreements.ErrAlreadyExists), errors.Is(err, agreements.ErrCompanyInUse):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "company operation failed")
	}
}
