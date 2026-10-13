package agreementdiscord

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/agreements"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/zap"
)

const (
	commandName   = "convenios"
	subConfigure  = "configurar"
	subStatus     = "estado"
	optionChannel = "canal"
	optionEnabled = "activo"
)

type command struct {
	service *agreements.Service
	log     *zap.Logger
}

func newCommand(service *agreements.Service, log *zap.Logger) discord.Command {
	handler := &command{service: service, log: log}
	return discord.Command{Definition: &discordgo.ApplicationCommand{Name: commandName, Description: "Gestiona el panel de convenios",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subConfigure, Description: "Define el canal de control y el estado",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionChannel, Name: optionChannel, Description: "Canal donde vive el panel de control",
						ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}},
					{Type: discordgo.ApplicationCommandOptionBoolean, Name: optionEnabled, Description: "Activa o desactiva los convenios"},
				}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subStatus, Description: "Muestra la configuración vigente"},
		}}, Handle: handler.handle}
}

func (handler *command) handle(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand || len(event.ApplicationCommandData().Options) == 0 ||
		discord.DeferEphemeral(ctx, session, event.Interaction) != nil {
		return
	}
	sub := event.ApplicationCommandData().Options[0]
	content := handler.status()
	if sub.Name == subConfigure {
		content = handler.configure(ctx, sub.Options)
	}
	if err := discord.EditResponse(ctx, session, event.Interaction, content, nil); err != nil {
		handler.log.Error("respond to agreements command", zap.Error(err))
	}
}

func (handler *command) configure(ctx context.Context, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	var update agreements.Update
	for _, option := range options {
		switch value := option.Value.(type) {
		case string:
			update.ControlChannelID = &value
		case bool:
			update.Enabled = &value
		}
	}
	if _, err := handler.service.Configure(ctx, update); err != nil {
		if errors.Is(err, agreements.ErrInvalidSettings) {
			return "❌ " + err.Error()
		}
		handler.log.Error("configure agreements", zap.Error(err))
		return "No fue posible guardar la configuración."
	}
	if err := handler.service.Publish(ctx); err != nil && !errors.Is(err, agreements.ErrDisabled) {
		handler.log.Error("publish agreements panel", zap.Error(err))
		return "⚠️ Configuración guardada, pero no se pudo publicar el panel; revisa los permisos del bot.\n\n" + handler.status()
	}
	return "✅ Configuración guardada.\n\n" + handler.status()
}

func (handler *command) status() string {
	config := handler.service.Current()
	state, channel := "inactivos ❌", "sin definir"
	if config.Enabled {
		state = "activos ✅"
	}
	if config.ControlChannelID != "" {
		channel = "<#" + config.ControlChannelID + ">"
	}
	return fmt.Sprintf("**Convenios:** %s\nCanal de control: %s\nRefresco automático: cada %s", state, channel, config.RefreshInterval)
}
