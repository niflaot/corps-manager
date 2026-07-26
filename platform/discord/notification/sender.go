// Package notification adapts durable verification notifications to Discord DMs.
package notification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/pixelados-net/discord-bot/internal/localization"
	"github.com/pixelados-net/discord-bot/internal/verification"
	verificationnotification "github.com/pixelados-net/discord-bot/internal/verification/notification"
	discordplatform "github.com/pixelados-net/discord-bot/platform/discord"
)

// Sender delivers localized verification transition DMs.
type Sender struct {
	// client owns the configured Discord session.
	client *discordplatform.Client
	// catalog resolves transition and group text.
	catalog *localization.Catalog
}

// NewSender creates a Discord verification notification sender.
func NewSender(client *discordplatform.Client, catalog *localization.Catalog) *Sender {
	return &Sender{client: client, catalog: catalog}
}

// Send delivers one Components V2 notification with a stable enforced nonce.
func (sender *Sender) Send(ctx context.Context, delivery verificationnotification.Delivery) (string, error) {
	channel, err := sender.client.SDK().UserChannelCreate(delivery.UserID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	components, err := sender.components(delivery)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(delivery.IdempotencyKey))
	nonce := hex.EncodeToString(digest[:])[:25]
	payload := struct {
		Components      []discordgo.MessageComponent      `json:"components"`
		AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions"`
		Flags           discordgo.MessageFlags            `json:"flags"`
		Nonce           string                            `json:"nonce"`
		EnforceNonce    bool                              `json:"enforce_nonce"`
	}{
		Components: components,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
		Flags: discordgo.MessageFlagsIsComponentsV2, Nonce: nonce, EnforceNonce: true,
	}
	endpoint := discordgo.EndpointChannelMessages(channel.ID)
	response, err := sender.client.SDK().RequestWithBucketID(
		http.MethodPost, endpoint, payload, endpoint, discordgo.WithContext(ctx),
	)
	if err != nil {
		return "", err
	}
	var message discordgo.Message
	if err := json.Unmarshal(response, &message); err != nil {
		return "", fmt.Errorf("decode verification notification response: %w", err)
	}
	if message.ID == "" {
		return "", fmt.Errorf("discord verification notification response omitted message ID")
	}
	return message.ID, nil
}

// components builds one localized Components V2 notification.
func (sender *Sender) components(delivery verificationnotification.Delivery) ([]discordgo.MessageComponent, error) {
	groupName := sender.catalog.GroupName(delivery.GroupKey)
	switch delivery.Kind {
	case verificationnotification.KindVerified:
		return []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: sender.catalog.Text(
				localization.VerificationSuccessKey, "group", groupName,
			)},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{
				Label: sender.catalog.Text(localization.VerificationUnverifyKey), Style: discordgo.DangerButton,
				CustomID: verification.LeaveCustomIDPrefix + delivery.GroupID,
			}}},
		}, nil
	case verificationnotification.KindUnverified:
		return []discordgo.MessageComponent{discordgo.TextDisplay{Content: sender.catalog.Text(
			localization.VerificationUnverifiedKey, "group", groupName,
		)}}, nil
	default:
		return nil, fmt.Errorf("unsupported verification notification kind %q", delivery.Kind)
	}
}
