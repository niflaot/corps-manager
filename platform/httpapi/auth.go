package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func authenticate(expected string) fiber.Handler {
	expectedDigest := sha256.Sum256([]byte(expected))
	return func(ctx *fiber.Ctx) error {
		scheme, token, found := strings.Cut(ctx.Get(fiber.HeaderAuthorization), " ")
		providedDigest := sha256.Sum256([]byte(token))
		if !found || !strings.EqualFold(scheme, "Bearer") || subtle.ConstantTimeCompare(expectedDigest[:], providedDigest[:]) != 1 {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		return ctx.Next()
	}
}
