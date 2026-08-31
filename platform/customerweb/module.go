// Package customerweb provides the public frequent-customer directory.
package customerweb

import (
	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/customers"
	"go.uber.org/fx"
)

// Module provides the named customer-page HTTP handler.
var Module = fx.Module("customer-web", fx.Provide(
	fx.Annotate(provideHandler, fx.ResultTags(`name:"customer_page"`)),
))

func provideHandler(service *customers.Service) fiber.Handler {
	page := New(service)
	return page.Handle
}
