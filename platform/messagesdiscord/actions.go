package messagesdiscord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/messages"
	"go.uber.org/zap"
)

// exportInlineLimit is the largest JSON export sent inline instead of as a file.
const exportInlineLimit = 1800

// listLimit bounds the managed messages shown by the list subcommand.
const listLimit = 25

func (c *command) simple(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, name string, item pending) {
	switch name {
	case subList:
		c.edit(ctx, session, interaction, c.list(ctx), nil)
	case subShow:
		c.show(ctx, session, interaction, item.Key)
	default:
		item.Action = actionArchive
		record, err := c.manager.Get(ctx, item.Key)
		if err != nil || record.State == messages.StateArchived {
			c.edit(ctx, session, interaction, c.describe(err, "El mensaje ya está archivado."), nil)
			return
		}
		id, err := c.pending.put(item)
		if err != nil {
			c.edit(ctx, session, interaction, "Hay demasiadas operaciones pendientes; inténtalo en unos minutos.", nil)
			return
		}
		c.edit(ctx, session, interaction, fmt.Sprintf("¿Archivar `%s`? Dejará de reconciliarse; el mensaje ya publicado en Discord no se borra.", item.Key), confirmRow(id))
	}
}

func (c *command) list(ctx context.Context) string {
	page, err := c.manager.List(ctx, messages.ListQuery{Limit: listLimit})
	if err != nil {
		return c.describe(err, "")
	}
	if len(page.Items) == 0 {
		return "No hay mensajes gestionados."
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "**Mensajes gestionados** (%d de %d)\n", len(page.Items), page.Total)
	for _, record := range page.Items {
		fmt.Fprintf(&builder, "`%s` · %s · <#%s> · rev %d\n", record.Key, record.State, record.ChannelID, record.Revision)
	}
	return builder.String()
}

func (c *command) show(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, key string) {
	record, err := c.manager.Get(ctx, key)
	if err != nil {
		c.edit(ctx, session, interaction, c.describe(err, ""), nil)
		return
	}
	encoded, err := json.MarshalIndent(record.Payload, "", "  ")
	if err != nil {
		c.edit(ctx, session, interaction, c.describe(err, ""), nil)
		return
	}
	header := fmt.Sprintf("`%s` · %s · <#%s> · rev %d", record.Key, record.State, record.ChannelID, record.Revision)
	if len(encoded) <= exportInlineLimit {
		c.edit(ctx, session, interaction, header+"\n```json\n"+string(encoded)+"\n```", nil)
		return
	}
	c.edit(ctx, session, interaction, header, nil, &discordgo.File{Name: key + ".json", ContentType: "application/json", Reader: strings.NewReader(string(encoded))})
}

func (c *command) execute(ctx context.Context, item pending, interactionID string) string {
	idempotency := "cmd:" + interactionID
	definition := messages.Definition{Key: item.Key, GuildID: c.guildID, ChannelID: item.ChannelID, Payload: item.Payload}
	var (
		result messages.MutationResult
		done   string
		err    error
	)
	switch item.Action {
	case actionCreate:
		result, err = c.manager.Create(ctx, definition, idempotency)
		done = "creado"
	case actionEdit, actionArchive:
		var record messages.Record
		if record, err = c.manager.Get(ctx, item.Key); err != nil {
			break
		}
		if item.Action == actionEdit {
			result, err = c.manager.Replace(ctx, item.Key, record.Revision, definition, idempotency)
			done = "actualizado"
		} else {
			result, err = c.manager.Archive(ctx, item.Key, record.Revision, idempotency)
			done = "archivado"
		}
	}
	if err != nil {
		return c.describe(err, "")
	}
	return fmt.Sprintf("✅ Mensaje `%s` %s (revisión %d). La publicación en Discord se sincroniza en segundos; revisa `/mensaje listar`.", item.Key, done, result.Record.Revision)
}

func (c *command) describe(err error, fallback string) string {
	switch {
	case err == nil:
		return fallback
	case errors.Is(err, messages.ErrNotFound):
		return "No existe un mensaje con esa clave."
	case errors.Is(err, messages.ErrConflict):
		return "La clave ya existe o el mensaje cambió mientras tanto; ejecuta el comando de nuevo."
	case errors.Is(err, messages.ErrInvalidAssignment):
		return "El canal no pertenece a este servidor."
	case errors.Is(err, messages.ErrForbidden):
		return "El bot no tiene permisos en ese canal."
	case errors.Is(err, messages.ErrInvalidDefinition):
		return "❌ " + truncate(strings.TrimPrefix(err.Error(), messages.ErrInvalidDefinition.Error()+": "))
	}
	c.log.Error("message command failed", zap.Error(err))
	return "No fue posible completar la operación; revisa los registros."
}
