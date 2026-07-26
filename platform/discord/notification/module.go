package notification

import (
	"github.com/pixelados-net/discord-bot/internal/localization"
	verificationnotification "github.com/pixelados-net/discord-bot/internal/verification/notification"
	discordplatform "github.com/pixelados-net/discord-bot/platform/discord"
	"go.uber.org/fx"
)

// Module provides durable verification notification delivery through Discord.
var Module = fx.Module("discord-verification-notification", fx.Provide(
	fx.Annotate(provideSender, fx.As(new(verificationnotification.Sender))),
))

// provideSender creates the Discord notification adapter.
func provideSender(client *discordplatform.Client, catalog *localization.Catalog) *Sender {
	return NewSender(client, catalog)
}
