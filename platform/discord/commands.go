package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const commandTimeout = 15 * time.Second

// noPermissionMessage is shown to members lacking the Administrator permission.
const noPermissionMessage = "Solo los administradores pueden usar este comando."

// CommandHandler processes every interaction routed to one command.
type CommandHandler func(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate)

// Command contributes one slash command and the follow-up interactions it owns.
type Command struct {
	// Definition is the Discord application command registered for the guild.
	Definition *discordgo.ApplicationCommand
	// ComponentPrefix routes component and modal interactions whose custom ID starts with it.
	ComponentPrefix string
	// Handle receives the command, its autocompletes, components, and modal submissions.
	Handle CommandHandler
}

// Router dispatches administrator-only interactions to contributed commands.
type Router struct {
	ctx      context.Context
	guildID  string
	commands []Command
	log      *zap.Logger
}

// NewRouter creates a router bound to one guild and a cancellable context.
func NewRouter(ctx context.Context, guildID string, commands []Command, log *zap.Logger) *Router {
	return &Router{ctx: ctx, guildID: guildID, commands: commands, log: log}
}

// Definitions returns the guild-only administrator-restricted command definitions.
func (router *Router) Definitions() []*discordgo.ApplicationCommand {
	permissions := int64(discordgo.PermissionAdministrator)
	definitions := make([]*discordgo.ApplicationCommand, 0, len(router.commands))
	for _, command := range router.commands {
		definition := *command.Definition
		definition.DefaultMemberPermissions = &permissions
		definition.Type = discordgo.ChatApplicationCommand
		definitions = append(definitions, &definition)
	}
	return definitions
}

// Handle routes one gateway interaction to its owning command.
func (router *Router) Handle(session *discordgo.Session, event *discordgo.InteractionCreate) {
	command, found := router.route(event)
	if !found || event.GuildID != router.guildID || event.Member == nil || event.Member.User == nil {
		return
	}
	if event.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		if event.Type != discordgo.InteractionApplicationCommandAutocomplete {
			_ = RespondEphemeral(router.ctx, session, event.Interaction, noPermissionMessage)
		}
		return
	}
	ctx, cancel := context.WithTimeout(router.ctx, commandTimeout)
	defer cancel()
	command.Handle(ctx, session, event)
}

func (router *Router) route(event *discordgo.InteractionCreate) (Command, bool) {
	for _, command := range router.commands {
		switch event.Type {
		case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
			if event.ApplicationCommandData().Name == command.Definition.Name {
				return command, true
			}
		case discordgo.InteractionMessageComponent:
			if command.ComponentPrefix != "" && strings.HasPrefix(event.MessageComponentData().CustomID, command.ComponentPrefix) {
				return command, true
			}
		case discordgo.InteractionModalSubmit:
			if command.ComponentPrefix != "" && strings.HasPrefix(event.ModalSubmitData().CustomID, command.ComponentPrefix) {
				return command, true
			}
		}
	}
	return Command{}, false
}

// Register overwrites the guild's slash commands with the contributed definitions.
func (router *Router) Register(ctx context.Context, client *Client) error {
	appID, err := client.BotUserID(ctx)
	if err != nil {
		return fmt.Errorf("resolve application id: %w", err)
	}
	created, err := client.SDK().ApplicationCommandBulkOverwrite(appID, router.guildID, router.Definitions(), discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("register slash commands: %w", err)
	}
	router.log.Info("slash commands registered", zap.Int("count", len(created)))
	return nil
}

// DeferEphemeral acknowledges an interaction with a private loading state.
func DeferEphemeral(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}, discordgo.WithContext(ctx))
}

// DeferUpdate acknowledges a component interaction that will edit its own message.
func DeferUpdate(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}, discordgo.WithContext(ctx))
}

// RespondEphemeral answers an interaction immediately with private text.
func RespondEphemeral(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, content string) error {
	return session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral,
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}},
	}, discordgo.WithContext(ctx))
}

// EditResponse replaces the interaction response text and components without mentions.
func EditResponse(ctx context.Context, session *discordgo.Session, interaction *discordgo.Interaction, content string, components []discordgo.MessageComponent, files ...*discordgo.File) error {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	_, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{Content: &content, Components: &components, Files: files,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
	}, discordgo.WithContext(ctx))
	return err
}
