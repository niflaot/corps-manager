package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/messages"
)

func registerMessageRoutes(router fiber.Router, service MessageService) {
	router.Post("/", createMessage(service))
	router.Get("/", listMessages(service))
	router.Get("/:key", getMessage(service))
	router.Put("/:key", replaceMessage(service))
	router.Put("/:key/assignment", assignMessage(service))
	router.Post("/:key/reconcile", reconcileMessage(service))
	router.Delete("/:key", archiveMessage(service))
}

func createMessage(service MessageService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var definition messages.Definition
		if err := decodeStrict(ctx.Body(), &definition); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		result, err := service.Create(ctx.UserContext(), definition, ctx.Get("Idempotency-Key"))
		if err != nil {
			return messageError(err)
		}
		status := fiber.StatusCreated
		if result.Replay {
			status = fiber.StatusOK
		}
		return ctx.Status(status).JSON(result.Record)
	}
}

func listMessages(service MessageService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		limit, err := parseOptionalInt(ctx.Query("limit"), 50)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid limit")
		}
		offset, err := parseOptionalInt(ctx.Query("offset"), 0)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid offset")
		}
		page, err := service.List(ctx.UserContext(), messages.ListQuery{State: messages.State(ctx.Query("state")), GuildID: ctx.Query("guildId"), ChannelID: ctx.Query("channelId"), Limit: limit, Offset: offset})
		if err != nil {
			return messageError(err)
		}
		return ctx.JSON(page)
	}
}

func getMessage(service MessageService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		record, err := service.Get(ctx.UserContext(), ctx.Params("key"))
		if err != nil {
			return messageError(err)
		}
		ctx.Set(fiber.HeaderETag, strconv.FormatUint(record.Revision, 10))
		return ctx.JSON(record)
	}
}

func replaceMessage(service MessageService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		revision, err := parseRevision(ctx.Get(fiber.HeaderIfMatch))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		var definition messages.Definition
		if err := decodeStrict(ctx.Body(), &definition); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		result, err := service.Replace(ctx.UserContext(), ctx.Params("key"), revision, definition, ctx.Get("Idempotency-Key"))
		if err != nil {
			return messageError(err)
		}
		return ctx.JSON(result.Record)
	}
}

func assignMessage(service MessageService) fiber.Handler {
	type assignment struct {
		GuildID               string `json:"guildId"`
		ChannelID             string `json:"channelId"`
		DeleteReplacedMessage bool   `json:"deleteReplacedMessage"`
	}
	return func(ctx *fiber.Ctx) error {
		revision, err := parseRevision(ctx.Get(fiber.HeaderIfMatch))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		var request assignment
		if err := decodeStrict(ctx.Body(), &request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if request.DeleteReplacedMessage {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "remote cleanup is not enabled")
		}
		record, err := service.Get(ctx.UserContext(), ctx.Params("key"))
		if err != nil {
			return messageError(err)
		}
		definition := record.Definition
		definition.GuildID, definition.ChannelID = request.GuildID, request.ChannelID
		result, err := service.Replace(ctx.UserContext(), record.Key, revision, definition, ctx.Get("Idempotency-Key"))
		if err != nil {
			return messageError(err)
		}
		return ctx.JSON(result.Record)
	}
}

func reconcileMessage(service MessageService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if err := service.Reconcile(ctx.UserContext(), ctx.Params("key")); err != nil {
			return messageError(err)
		}
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "pending"})
	}
}

func archiveMessage(service MessageService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		revision, err := parseRevision(ctx.Get(fiber.HeaderIfMatch))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		result, err := service.Archive(ctx.UserContext(), ctx.Params("key"), revision, ctx.Get("Idempotency-Key"))
		if err != nil {
			return messageError(err)
		}
		return ctx.JSON(result.Record)
	}
}

func decodeStrict(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid JSON: multiple values")
	}
	return nil
}

func parseRevision(raw string) (uint64, error) {
	raw = strings.Trim(strings.TrimSpace(raw), `"`)
	revision, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || revision == 0 {
		return 0, fmt.Errorf("If-Match revision is required")
	}
	return revision, nil
}

func parseOptionalInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func messageError(err error) error {
	switch {
	case errors.Is(err, messages.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, messages.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, messages.ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, messages.ErrRateLimited):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	case errors.Is(err, messages.ErrInvalidDefinition):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal server error")
	}
}
