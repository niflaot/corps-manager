package verification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/internal/settings"
)

// GuardGateway reconciles Discord channel safety controls.
type GuardGateway interface {
	// MakeReadOnly denies member writes in one channel.
	MakeReadOnly(context.Context, string) error
	// ReconcileTrap ensures the visible writable last channel and warning message.
	ReconcileTrap(context.Context, string, string, string, string) (string, string, error)
}

// MessageManager supplies managed-message use cases needed by the guard.
type MessageManager interface {
	// Get returns one managed message.
	Get(context.Context, string) (messages.Record, error)
	// Replace applies an idempotent desired-state replacement.
	Replace(context.Context, string, uint64, messages.Definition, string) (messages.MutationResult, error)
}

// Guard reconciles verification markup, channel permissions, and trap state.
type Guard struct {
	groups    Repository
	messages  MessageManager
	settings  *settings.Service
	gateway   GuardGateway
	lock      sync.Mutex
	emptyText string
}

// NewGuard creates the verification guard reconciler.
func NewGuard(groups Repository, messageManager MessageManager, settingService *settings.Service, gateway GuardGateway, emptyText ...string) *Guard {
	fallback := "Verification unavailable"
	if len(emptyText) > 0 && strings.TrimSpace(emptyText[0]) != "" {
		fallback = emptyText[0]
	}
	return &Guard{groups: groups, messages: messageManager, settings: settingService, gateway: gateway, emptyText: fallback}
}

// Reconcile repairs all verification guard desired state.
func (guard *Guard) Reconcile(ctx context.Context) error {
	if !guard.lock.TryLock() {
		return nil
	}
	defer guard.lock.Unlock()
	messageError := guard.reconcileMessage(ctx)
	name, err := guard.settings.String(ctx, settings.VerificationTrapChannelName)
	if err != nil {
		return err
	}
	warning, err := guard.settings.String(ctx, settings.VerificationTrapWarning)
	if err != nil {
		return err
	}
	channel, err := guard.settings.Get(ctx, settings.VerificationTrapChannelID)
	if err != nil {
		return err
	}
	message, err := guard.settings.Get(ctx, settings.VerificationTrapMessageID)
	if err != nil {
		return err
	}
	channelID, messageID := stringValue(channel.Value), stringValue(message.Value)
	channelID, messageID, err = guard.gateway.ReconcileTrap(ctx, name, warning, channelID, messageID)
	if err != nil {
		return err
	}
	if channelID != stringValue(channel.Value) {
		if _, err = guard.settings.Set(ctx, settings.VerificationTrapChannelID, jsonString(channelID), channel.Revision); err != nil {
			return err
		}
	}
	if messageID != stringValue(message.Value) {
		if _, err = guard.settings.Set(ctx, settings.VerificationTrapMessageID, jsonString(messageID), message.Revision); err != nil {
			return err
		}
	}
	return messageError
}

func (guard *Guard) reconcileMessage(ctx context.Context) error {
	key, err := guard.settings.String(ctx, settings.VerificationMessageKey)
	if err != nil {
		return err
	}
	record, err := guard.messages.Get(ctx, key)
	if err != nil {
		return err
	}
	groups, err := guard.groups.ListGroups(ctx, true)
	if err != nil {
		return err
	}
	components := removeVerificationRows(record.Payload.Components)
	if len(components) == 0 {
		text, marshalErr := messages.DecodeComponent(discordgo.TextDisplay{Content: guard.emptyText})
		if marshalErr != nil {
			return marshalErr
		}
		components = append(components, text)
	}
	if len(groups) > 0 {
		buttons := make([]discordgo.MessageComponent, len(groups))
		for index, group := range groups {
			button := discordgo.Button{Label: group.ButtonLabel, Style: discordgo.ButtonStyle(group.ButtonStyle), CustomID: JoinCustomIDPrefix + group.ID}
			if group.ButtonEmoji != "" {
				button.Emoji = &discordgo.ComponentEmoji{Name: group.ButtonEmoji}
			}
			buttons[index] = button
		}
		row, marshalErr := messages.DecodeComponent(discordgo.ActionsRow{Components: buttons})
		if marshalErr != nil {
			return marshalErr
		}
		components = append(components, row)
	}
	definition := record.Definition
	definition.Payload.Components = components
	if err := definition.Payload.Validate(); err != nil {
		return err
	}
	desired, _ := definition.Payload.Hash()
	if desired != record.DesiredHash {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", key, record.Revision, desired)))
		if _, err := guard.messages.Replace(ctx, key, record.Revision, definition, "verification-"+hex.EncodeToString(digest[:])); err != nil {
			return err
		}
	}
	return guard.gateway.MakeReadOnly(ctx, record.ChannelID)
}

func removeVerificationRows(components []messages.Component) []messages.Component {
	result := make([]messages.Component, 0, len(components))
	for _, raw := range components {
		component, err := discordgo.MessageComponentFromJSON(raw)
		if err != nil {
			continue
		}
		row, ok := component.(*discordgo.ActionsRow)
		remove := false
		if ok {
			for _, child := range row.Components {
				if button, buttonOK := child.(*discordgo.Button); buttonOK && strings.HasPrefix(button.CustomID, JoinCustomIDPrefix) {
					remove = true
				}
			}
		}
		if !remove {
			result = append(result, raw)
		}
	}
	return result
}

func jsonString(value string) json.RawMessage { encoded, _ := json.Marshal(value); return encoded }
func stringValue(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
