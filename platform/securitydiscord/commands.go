package securitydiscord

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/internal/security"
	"github.com/niflaot/corps-manager/platform/discord"
	"go.uber.org/zap"
)

const (
	verificationCommand = "verificacion"
	antibotCommand      = "antibot"
	subConfigure        = "configurar"
	subPublish          = "publicar"
	subStatus           = "estado"
	optionChannel       = "canal"
	optionRole          = "rol"
	optionKey           = "clave"
	optionEnabled       = "activo"
)

type commands struct {
	service *security.Service
	log     *zap.Logger
}

func newVerificationCommand(service *security.Service, log *zap.Logger) discord.Command {
	handler := &commands{service: service, log: log}
	return discord.Command{Definition: &discordgo.ApplicationCommand{Name: verificationCommand, Description: "Gestiona la verificación de miembros",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subConfigure, Description: "Define canal, rol y estado de la verificación",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionChannel, Name: optionChannel, Description: "Canal del mensaje de verificación",
						ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews}},
					{Type: discordgo.ApplicationCommandOptionRole, Name: optionRole, Description: "Rol que se otorga al verificarse"},
					{Type: discordgo.ApplicationCommandOptionString, Name: optionKey, Description: "Clave del mensaje gestionado", MaxLength: 64},
					{Type: discordgo.ApplicationCommandOptionBoolean, Name: optionEnabled, Description: "Activa o desactiva la verificación"},
				}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subPublish, Description: "Publica o repara el mensaje de verificación"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subStatus, Description: "Muestra la configuración vigente"},
		}}, Handle: handler.verification}
}

func newAntibotCommand(service *security.Service, log *zap.Logger) discord.Command {
	handler := &commands{service: service, log: log}
	return discord.Command{Definition: &discordgo.ApplicationCommand{Name: antibotCommand, Description: "Gestiona la trampa anti bots",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subConfigure, Description: "Activa o desactiva la trampa",
				Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionBoolean, Name: optionEnabled, Description: "Estado deseado", Required: true}}},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subStatus, Description: "Muestra el estado de la trampa"},
		}}, Handle: handler.antibot}
}

func (handler *commands) verification(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand || len(event.ApplicationCommandData().Options) == 0 || discord.DeferEphemeral(ctx, session, event.Interaction) != nil {
		return
	}
	sub := event.ApplicationCommandData().Options[0]
	content := ""
	switch sub.Name {
	case subConfigure:
		content = handler.configure(ctx, handler.verificationUpdate(sub.Options))
	case subPublish:
		content = handler.publish(ctx)
	default:
		content = handler.status(ctx)
	}
	handler.reply(ctx, session, event, content)
}

func (handler *commands) antibot(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	if event.Type != discordgo.InteractionApplicationCommand || len(event.ApplicationCommandData().Options) == 0 || discord.DeferEphemeral(ctx, session, event.Interaction) != nil {
		return
	}
	sub := event.ApplicationCommandData().Options[0]
	content := ""
	if sub.Name == subConfigure {
		enabled := optionValue(sub.Options, optionEnabled) == "true"
		content = handler.configure(ctx, security.Update{AntibotEnabled: &enabled})
	} else {
		content = handler.status(ctx)
	}
	handler.reply(ctx, session, event, content)
}

func (handler *commands) reply(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate, content string) {
	if err := discord.EditResponse(ctx, session, event.Interaction, content, nil); err != nil {
		handler.log.Error("respond to security command", zap.Error(err))
	}
}

func (handler *commands) verificationUpdate(options []*discordgo.ApplicationCommandInteractionDataOption) security.Update {
	var update security.Update
	if value := optionValue(options, optionChannel); value != "" {
		update.ChannelID = &value
	}
	if value := optionValue(options, optionRole); value != "" {
		update.RoleID = &value
	}
	if value := optionValue(options, optionKey); value != "" {
		update.MessageKey = &value
	}
	if value := optionValue(options, optionEnabled); value != "" {
		enabled := value == "true"
		update.VerificationEnabled = &enabled
	}
	return update
}

func (handler *commands) configure(ctx context.Context, update security.Update) string {
	if _, err := handler.service.Configure(ctx, update); err != nil {
		if errors.Is(err, security.ErrInvalidSettings) {
			return "❌ " + err.Error()
		}
		handler.log.Error("configure security", zap.Error(err))
		return "No fue posible guardar la configuración."
	}
	if err := handler.service.Refresh(ctx); err != nil {
		handler.log.Error("apply security settings", zap.Error(err))
		return "⚠️ Configuración guardada, pero no se pudo aplicar:\n" + handler.describeRefresh(err) + "\n\n" + handler.status(ctx)
	}
	return "✅ Configuración guardada.\n\n" + handler.status(ctx)
}

func (handler *commands) publish(ctx context.Context) string {
	if err := handler.service.Refresh(ctx); err != nil {
		handler.log.Error("publish verification", zap.Error(err))
		return "⚠️ " + handler.describeRefresh(err)
	}
	return "✅ Verificación sincronizada.\n\n" + handler.status(ctx)
}

func (handler *commands) describeRefresh(err error) string {
	if errors.Is(err, messages.ErrNotFound) {
		return "No existe el mensaje gestionado. Créalo con `/mensaje crear` usando la clave configurada."
	}
	return "Revisa los permisos del bot y el estado del canal."
}

func (handler *commands) status(ctx context.Context) string {
	status, err := handler.service.Status(ctx)
	if err != nil {
		handler.log.Error("read security status", zap.Error(err))
		return "No fue posible leer el estado."
	}
	config := status.Config
	message := "no existe (créalo con `/mensaje crear clave:" + config.MessageKey + "`)"
	if status.MessageFound {
		message = fmt.Sprintf("`%s` · estado `%s` · revisión %d", config.MessageKey, status.Message.State, status.Message.Revision)
	}
	trap := "sin canal activo"
	if status.TrapChannelID != "" {
		trap = "<#" + status.TrapChannelID + ">"
	}
	return fmt.Sprintf("**Verificación:** %s\nCanal: %s\nRol: %s\nMensaje: %s\n\n**Anti bots:** %s\nTrampa: %s",
		toggle(config.VerificationEnabled), mention("#", config.ChannelID), mention("@&", config.RoleID), message, toggle(config.AntibotEnabled), trap)
}

func toggle(enabled bool) string {
	if enabled {
		return "activa ✅"
	}
	return "inactiva ❌"
}

func mention(prefix, id string) string {
	if id == "" {
		return "sin definir"
	}
	return "<" + prefix + id + ">"
}

func optionValue(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if option.Name != name {
			continue
		}
		switch value := option.Value.(type) {
		case string:
			return value
		case bool:
			return fmt.Sprint(value)
		}
	}
	return ""
}
