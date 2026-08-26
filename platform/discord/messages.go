package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/messages"
)

// MessageGateway adapts the configured Discord session to managed messages.
type MessageGateway struct {
	client  *Client
	guildID string
}

// NewMessageGateway creates a guild-scoped managed-message Discord adapter.
func NewMessageGateway(client *Client, guildID string) *MessageGateway {
	return &MessageGateway{client: client, guildID: guildID}
}

// ValidateAssignment verifies that one channel belongs to the configured guild.
func (gateway *MessageGateway) ValidateAssignment(ctx context.Context, guildID string, channelID string) error {
	if guildID != gateway.guildID {
		return messages.ErrInvalidAssignment
	}
	channel, err := gateway.client.session.Channel(channelID, discordgo.WithContext(ctx))
	if err != nil {
		mapped := mapMessageError(err)
		if errors.Is(mapped, messages.ErrNotFound) {
			return messages.ErrInvalidAssignment
		}
		return mapped
	}
	if channel.GuildID != gateway.guildID {
		return messages.ErrInvalidAssignment
	}
	return nil
}

// Get reads one Discord message and its ownership.
func (gateway *MessageGateway) Get(ctx context.Context, channelID string, messageID string) (messages.ObservedMessage, error) {
	message, err := gateway.client.session.ChannelMessage(channelID, messageID, discordgo.WithContext(ctx))
	if err != nil {
		return messages.ObservedMessage{}, mapMessageError(err)
	}
	return gateway.observe(ctx, message)
}

// Create sends one Components V2 message with an enforced Discord nonce.
func (gateway *MessageGateway) Create(ctx context.Context, request messages.CreateRequest) (messages.ObservedMessage, error) {
	components, err := toDiscordComponents(request.Payload.Components)
	if err != nil {
		return messages.ObservedMessage{}, fmt.Errorf("%w: %v", messages.ErrInvalidRemote, err)
	}
	payload := struct {
		Components      []discordgo.MessageComponent      `json:"components"`
		AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions"`
		Flags           discordgo.MessageFlags            `json:"flags"`
		Nonce           string                            `json:"nonce"`
		EnforceNonce    bool                              `json:"enforce_nonce"`
	}{components, toDiscordMentions(request.Payload.AllowedMentions), discordgo.MessageFlagsIsComponentsV2, request.Nonce, true}
	endpoint := discordgo.EndpointChannelMessages(request.ChannelID)
	response, err := gateway.client.session.RequestWithBucketID(http.MethodPost, endpoint, payload, endpoint, discordgo.WithContext(ctx))
	if err != nil {
		mapped := mapMessageError(err)
		if errors.Is(mapped, messages.ErrForbidden) || errors.Is(mapped, messages.ErrInvalidRemote) || errors.Is(mapped, messages.ErrRateLimited) || errors.Is(mapped, messages.ErrNotFound) {
			return messages.ObservedMessage{}, mapped
		}
		return messages.ObservedMessage{}, fmt.Errorf("%w: %v", messages.ErrAmbiguousCreate, err)
	}
	var created discordgo.Message
	if err := json.Unmarshal(response, &created); err != nil {
		return messages.ObservedMessage{}, fmt.Errorf("%w: decode Discord create response", messages.ErrAmbiguousCreate)
	}
	return gateway.observe(ctx, &created)
}

// Replace completely replaces managed Components V2 state.
func (gateway *MessageGateway) Replace(ctx context.Context, request messages.ReplaceRequest) (messages.ObservedMessage, error) {
	components, err := toDiscordComponents(request.Payload.Components)
	if err != nil {
		return messages.ObservedMessage{}, fmt.Errorf("%w: %v", messages.ErrInvalidRemote, err)
	}
	edit := &discordgo.MessageEdit{ID: request.MessageID, Channel: request.ChannelID, Components: &components,
		AllowedMentions: toDiscordMentions(request.Payload.AllowedMentions), Flags: discordgo.MessageFlagsIsComponentsV2}
	message, err := gateway.client.session.ChannelMessageEditComplex(edit, discordgo.WithContext(ctx))
	if err != nil {
		return messages.ObservedMessage{}, mapMessageError(err)
	}
	return gateway.observe(ctx, message)
}

// Delete removes one managed Discord message.
func (gateway *MessageGateway) Delete(ctx context.Context, channelID string, messageID string) error {
	return mapMessageError(gateway.client.session.ChannelMessageDelete(channelID, messageID, discordgo.WithContext(ctx)))
}

func (gateway *MessageGateway) observe(ctx context.Context, message *discordgo.Message) (messages.ObservedMessage, error) {
	userID, err := gateway.client.BotUserID(ctx)
	if err != nil {
		return messages.ObservedMessage{}, err
	}
	components := make([]messages.Component, len(message.Components))
	for index, component := range message.Components {
		components[index], err = messages.DecodeComponent(component)
		if err != nil {
			return messages.ObservedMessage{}, err
		}
	}
	authorID := ""
	if message.Author != nil {
		authorID = message.Author.ID
	}
	return messages.ObservedMessage{ID: message.ID, GuildID: message.GuildID, ChannelID: message.ChannelID,
		Owned: authorID == userID, ComponentsV2: message.Flags&discordgo.MessageFlagsIsComponentsV2 != 0,
		Payload: messages.Payload{Components: components, AllowedMentions: messages.AllowedMentions{Parse: []string{}}}.Normalize()}, nil
}

func toDiscordComponents(raw []messages.Component) ([]discordgo.MessageComponent, error) {
	components := make([]discordgo.MessageComponent, len(raw))
	for index := range raw {
		component, err := discordgo.MessageComponentFromJSON(raw[index])
		if err != nil {
			return nil, err
		}
		components[index] = component
	}
	return components, nil
}

func toDiscordMentions(mentions messages.AllowedMentions) *discordgo.MessageAllowedMentions {
	parse := make([]discordgo.AllowedMentionType, len(mentions.Parse))
	for index, item := range mentions.Parse {
		parse[index] = discordgo.AllowedMentionType(item)
	}
	return &discordgo.MessageAllowedMentions{Parse: parse, Roles: mentions.Roles, Users: mentions.Users, RepliedUser: mentions.RepliedUser}
}

func mapMessageError(err error) error {
	if err == nil {
		return nil
	}
	var restError *discordgo.RESTError
	if errors.As(err, &restError) && restError.Response != nil {
		switch restError.Response.StatusCode {
		case http.StatusBadRequest:
			return messages.ErrInvalidRemote
		case http.StatusForbidden:
			return messages.ErrForbidden
		case http.StatusNotFound:
			return messages.ErrNotFound
		case http.StatusTooManyRequests:
			return messages.ErrRateLimited
		}
	}
	return err
}
