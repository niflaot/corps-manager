package httpapi

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/verification"
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
	if dependencies.Guild != nil {
		application.Get("/guild/members", getGuildMemberCount(dependencies.Guild))
	}
	if dependencies.DiscordLinks != nil {
		dependencies.DiscordLinks.RegisterPublic(application)
		dependencies.DiscordLinks.RegisterPrivate(application.Group("/api/discord-links", authenticate(apiConfig.APIKey)))
	}
	if dependencies.Messages != nil {
		registerMessageRoutes(application.Group("/api/messages", authenticate(apiConfig.APIKey)), dependencies.Messages)
	}
	if dependencies.Settings != nil {
		registerSettingRoutes(application.Group("/api/settings", authenticate(apiConfig.APIKey)), dependencies.Settings)
	}
	if dependencies.Verification != nil {
		registerVerificationRoutes(application.Group("/api/verification", authenticate(apiConfig.APIKey)), dependencies.Verification, dependencies.VerificationGuard)
	}
	application.Use(func(*fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "route not found")
	})
}

func getGuildMemberCount(service GuildService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		memberCount, presenceCount, err := service.MemberCount(ctx.UserContext())
		if err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "guild statistics unavailable")
		}
		return ctx.JSON(GuildStats{MemberCount: memberCount, PresenceCount: presenceCount})
	}
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

func registerVerificationRoutes(router fiber.Router, service VerificationService, guard VerificationGuard) {
	router.Post("/groups", createVerificationGroup(service))
	router.Get("/groups", listVerificationGroups(service))
	router.Get("/groups/:id", getVerificationGroup(service))
	router.Put("/groups/:id", updateVerificationGroup(service))
	router.Delete("/groups/:id", deleteVerificationGroup(service))
	router.Get("/memberships", listVerificationMemberships(service))
	if guard != nil {
		router.Post("/reconcile", reconcileVerification(service, guard))
	}
}

func createVerificationGroup(service VerificationService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var group verification.Group
		if err := decodeStrict(ctx.Body(), &group); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		created, err := service.CreateGroup(ctx.UserContext(), group)
		if err != nil {
			return verificationError(err)
		}
		return ctx.Status(fiber.StatusCreated).JSON(created)
	}
}

func listVerificationGroups(service VerificationService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		groups, err := service.ListGroups(ctx.UserContext(), ctx.QueryBool("enabledOnly", false))
		if err != nil {
			return verificationError(err)
		}
		return ctx.JSON(fiber.Map{"items": groups})
	}
}

func getVerificationGroup(service VerificationService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		group, err := service.GetGroup(ctx.UserContext(), ctx.Params("id"))
		if err != nil {
			return verificationError(err)
		}
		ctx.Set(fiber.HeaderETag, strconv.FormatUint(group.Revision, 10))
		return ctx.JSON(group)
	}
}

func updateVerificationGroup(service VerificationService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		revision, err := parseRevision(ctx.Get(fiber.HeaderIfMatch))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		var group verification.Group
		if err = decodeStrict(ctx.Body(), &group); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		updated, err := service.UpdateGroup(ctx.UserContext(), ctx.Params("id"), revision, group)
		if err != nil {
			return verificationError(err)
		}
		return ctx.JSON(updated)
	}
}

func deleteVerificationGroup(service VerificationService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		revision, err := parseRevision(ctx.Get(fiber.HeaderIfMatch))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err = service.DeleteGroup(ctx.UserContext(), ctx.Params("id"), revision); err != nil {
			return verificationError(err)
		}
		return ctx.SendStatus(fiber.StatusNoContent)
	}
}

func listVerificationMemberships(service VerificationService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		page, err := service.ListMemberships(ctx.UserContext(), ctx.Query("userId"))
		if err != nil {
			return verificationError(err)
		}
		return ctx.JSON(page)
	}
}

func reconcileVerification(service VerificationService, guard VerificationGuard) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		guardError := guard.Reconcile(ctx.UserContext())
		membershipError := service.ReconcileMemberships(ctx.UserContext())
		if err := errors.Join(guardError, membershipError); err != nil {
			return verificationError(err)
		}
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "reconciled"})
	}
}

func verificationError(err error) error {
	switch {
	case errors.Is(err, verification.ErrInvalid):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, verification.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, verification.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal server error")
	}
}
