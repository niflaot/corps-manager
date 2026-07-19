package linkapi

import (
	"errors"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

type createIntentResponse struct {
	discordlinks.Intent
	StartURL string `json:"startUrl"`
}

type exchangeRequest struct {
	Code string `json:"code"`
}

func (routes *Routes) createIntent(ctx *fiber.Ctx) error {
	var request discordlinks.CreateIntent
	if err := decodeStrict(ctx.Body(), &request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if _, ok := routes.config.CompletionURLs[request.CompletionKey]; !ok {
		return fiber.NewError(fiber.StatusBadRequest, "unknown completion key")
	}
	request.IdempotencyKey = ctx.Get("Idempotency-Key")
	intent, err := routes.service.CreateIntent(ctx.UserContext(), request)
	if err != nil {
		return linkError(err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(createIntentResponse{Intent: intent,
		StartURL: routes.config.startURL(intent.ID)})
}

func (routes *Routes) createLoginIntent(ctx *fiber.Ctx) error {
	var request discordlinks.CreateLoginIntent
	if err := decodeStrict(ctx.Body(), &request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if _, ok := routes.config.CompletionURLs[request.CompletionKey]; !ok {
		return fiber.NewError(fiber.StatusBadRequest, "unknown completion key")
	}
	request.IdempotencyKey = ctx.Get("Idempotency-Key")
	intent, err := routes.service.CreateLoginIntent(ctx.UserContext(), request)
	if err != nil {
		return linkError(err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(createIntentResponse{Intent: intent,
		StartURL: routes.config.startURL(intent.ID)})
}

func (routes *Routes) start(ctx *fiber.Ctx) error {
	destination, err := routes.service.Start(ctx.UserContext(), ctx.Params("intentId"))
	if err != nil {
		return linkError(err)
	}
	return ctx.Redirect(destination, fiber.StatusFound)
}

func (routes *Routes) callback(ctx *fiber.Ctx) error {
	completion, err := routes.service.Complete(ctx.UserContext(), discordlinks.Callback{
		State: ctx.Query("state"), Code: ctx.Query("code"), ProviderError: ctx.Query("error"),
	})
	if err != nil {
		return linkError(err)
	}
	destination, err := routes.config.completionURL(completion.CompletionKey, completion.Code)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "completion destination unavailable")
	}
	return ctx.Redirect(destination, fiber.StatusSeeOther)
}

func (routes *Routes) exchangeResult(ctx *fiber.Ctx) error {
	var request exchangeRequest
	if err := decodeStrict(ctx.Body(), &request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	result, err := routes.service.ExchangeResult(ctx.UserContext(), request.Code, ctx.Get("Idempotency-Key"))
	if err != nil {
		return linkError(err)
	}
	return ctx.JSON(result)
}

func (routes *Routes) getBySubject(ctx *fiber.Ctx) error {
	subject, err := url.PathUnescape(ctx.Params("subject"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid subject")
	}
	link, err := routes.service.LinkBySubject(ctx.UserContext(), subject)
	if err != nil {
		return linkError(err)
	}
	return ctx.JSON(link)
}

func (routes *Routes) getByDiscordUser(ctx *fiber.Ctx) error {
	link, err := routes.service.LinkByDiscordUser(ctx.UserContext(), ctx.Params("discordUserId"))
	if err != nil {
		return linkError(err)
	}
	return ctx.JSON(link)
}

func (routes *Routes) unlink(ctx *fiber.Ctx) error {
	subject, err := url.PathUnescape(ctx.Params("subject"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid subject")
	}
	link, err := routes.service.Unlink(ctx.UserContext(), strings.TrimSpace(subject))
	if err != nil {
		return linkError(err)
	}
	return ctx.JSON(link)
}

func linkError(err error) error {
	switch {
	case errors.Is(err, discordlinks.ErrInvalid):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, discordlinks.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, discordlinks.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, discordlinks.ErrExpired), errors.Is(err, discordlinks.ErrGone):
		return fiber.NewError(fiber.StatusGone, err.Error())
	case errors.Is(err, discordlinks.ErrProvider):
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	case errors.Is(err, discordlinks.ErrUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal server error")
	}
}
