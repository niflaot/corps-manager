package linkapi

import (
	"context"
	"regexp"

	"github.com/gofiber/fiber/v2"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

var completionKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type service interface {
	CreateIntent(context.Context, discordlinks.CreateIntent) (discordlinks.Intent, error)
	CreateLoginIntent(context.Context, discordlinks.CreateLoginIntent) (discordlinks.Intent, error)
	Start(context.Context, string) (string, error)
	Complete(context.Context, discordlinks.Callback) (discordlinks.Completion, error)
	ExchangeResult(context.Context, string, string) (discordlinks.Result, error)
	LinkBySubject(context.Context, string) (discordlinks.Link, error)
	LinkByDiscordUser(context.Context, string) (discordlinks.Link, error)
	Unlink(context.Context, string) (discordlinks.Link, error)
}

// Routes owns public OAuth redirects and private link-management handlers.
type Routes struct {
	service service
	config  Config
}

// New creates Discord account-link HTTP routes.
func New(service service, config Config) *Routes {
	return &Routes{service: service, config: config}
}

// RegisterPublic registers browser endpoints that cannot require an API key.
func (routes *Routes) RegisterPublic(router fiber.Router) {
	router.Get("/oauth/discord/start/:intentId", routes.start)
	router.Get("/oauth/discord/callback", routes.callback)
}

// RegisterPrivate registers service endpoints behind the caller's API-key middleware.
func (routes *Routes) RegisterPrivate(router fiber.Router) {
	router.Post("/intents", routes.createIntent)
	router.Post("/login-intents", routes.createLoginIntent)
	router.Post("/results/exchange", routes.exchangeResult)
	router.Get("/subjects/:subject", routes.getBySubject)
	router.Delete("/subjects/:subject", routes.unlink)
	router.Get("/users/:discordUserId", routes.getByDiscordUser)
}
