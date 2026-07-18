package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/pixelados-net/discord-bot/internal/messages"
)

// MessageGateway adapts the configured Discord session to managed messages.
type MessageGateway struct {
	client *Client
}

// NewMessageGateway creates a managed-message Discord adapter.
func NewMessageGateway(client *Client) *MessageGateway {
	return &MessageGateway{client: client}
}

// ValidateAssignment verifies that one channel belongs to the desired guild.
func (gateway *MessageGateway) ValidateAssignment(ctx context.Context, guildID string, channelID string) error {
	channel, err := gateway.client.session.Channel(channelID, discordgo.WithContext(ctx))
	if err != nil {
		mapped := mapMessageError(err)
		if errors.Is(mapped, messages.ErrNotFound) {
			return messages.ErrInvalidAssignment
		}
		return mapped
	}
	if channel.GuildID != guildID {
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

// Create sends one message with an enforced Discord nonce.
func (gateway *MessageGateway) Create(ctx context.Context, request messages.CreateRequest) (messages.ObservedMessage, error) {
	payload := struct {
		Content         string                            `json:"content,omitempty"`
		Embeds          []*discordgo.MessageEmbed         `json:"embeds"`
		AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions"`
		Nonce           string                            `json:"nonce"`
		EnforceNonce    bool                              `json:"enforce_nonce"`
	}{
		Content:         request.Payload.Content,
		Embeds:          toDiscordEmbeds(request.Payload.Embeds),
		AllowedMentions: toDiscordMentions(request.Payload.AllowedMentions),
		Nonce:           request.Nonce,
		EnforceNonce:    true,
	}
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
	observed, err := gateway.observe(ctx, &created)
	if err != nil {
		return messages.ObservedMessage{}, fmt.Errorf("%w: verify Discord create response", messages.ErrAmbiguousCreate)
	}
	return observed, nil
}

// Replace completely replaces managed content and embeds.
func (gateway *MessageGateway) Replace(ctx context.Context, request messages.ReplaceRequest) (messages.ObservedMessage, error) {
	content := request.Payload.Content
	embeds := toDiscordEmbeds(request.Payload.Embeds)
	edit := &discordgo.MessageEdit{
		ID:              request.MessageID,
		Channel:         request.ChannelID,
		Content:         &content,
		Embeds:          &embeds,
		AllowedMentions: toDiscordMentions(request.Payload.AllowedMentions),
	}
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
	authorID := ""
	if message.Author != nil {
		authorID = message.Author.ID
	}
	return messages.ObservedMessage{
		ID: message.ID, GuildID: message.GuildID, ChannelID: message.ChannelID, Owned: authorID == userID,
		Payload: messages.Payload{Content: message.Content, Embeds: fromDiscordEmbeds(message.Embeds), AllowedMentions: messages.AllowedMentions{Parse: []string{}}}.Normalize(),
	}, nil
}

func toDiscordEmbeds(embeds []messages.Embed) []*discordgo.MessageEmbed {
	result := make([]*discordgo.MessageEmbed, len(embeds))
	for index, embed := range embeds {
		converted := &discordgo.MessageEmbed{Title: embed.Title, Description: embed.Description, URL: embed.URL, Timestamp: embed.Timestamp, Color: embed.Color}
		if embed.Author != nil {
			converted.Author = &discordgo.MessageEmbedAuthor{Name: embed.Author.Name, URL: embed.Author.URL, IconURL: embed.Author.IconURL}
		}
		if embed.Footer != nil {
			converted.Footer = &discordgo.MessageEmbedFooter{Text: embed.Footer.Text, IconURL: embed.Footer.IconURL}
		}
		if embed.Image != nil {
			converted.Image = &discordgo.MessageEmbedImage{URL: embed.Image.URL}
		}
		if embed.Thumbnail != nil {
			converted.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: embed.Thumbnail.URL}
		}
		converted.Fields = make([]*discordgo.MessageEmbedField, len(embed.Fields))
		for fieldIndex, field := range embed.Fields {
			converted.Fields[fieldIndex] = &discordgo.MessageEmbedField{Name: field.Name, Value: field.Value, Inline: field.Inline}
		}
		result[index] = converted
	}
	return result
}

func fromDiscordEmbeds(embeds []*discordgo.MessageEmbed) []messages.Embed {
	result := make([]messages.Embed, len(embeds))
	for index, embed := range embeds {
		converted := messages.Embed{Title: embed.Title, Description: embed.Description, URL: embed.URL, Timestamp: embed.Timestamp, Color: embed.Color, Fields: []messages.EmbedField{}}
		if embed.Author != nil {
			converted.Author = &messages.EmbedAuthor{Name: embed.Author.Name, URL: embed.Author.URL, IconURL: embed.Author.IconURL}
		}
		if embed.Footer != nil {
			converted.Footer = &messages.EmbedFooter{Text: embed.Footer.Text, IconURL: embed.Footer.IconURL}
		}
		if embed.Image != nil {
			converted.Image = &messages.EmbedMedia{URL: embed.Image.URL}
		}
		if embed.Thumbnail != nil {
			converted.Thumbnail = &messages.EmbedMedia{URL: embed.Thumbnail.URL}
		}
		for _, field := range embed.Fields {
			converted.Fields = append(converted.Fields, messages.EmbedField{Name: field.Name, Value: field.Value, Inline: field.Inline})
		}
		result[index] = converted
	}
	return result
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
	var rateLimit discordgo.RateLimitError
	if errors.As(err, &rateLimit) {
		return messages.ErrRateLimited
	}
	return err
}
