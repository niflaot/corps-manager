package security

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/internal/settings"
)

var (
	// ErrInvalidVerification rejects interactions outside the configured message.
	ErrInvalidVerification = errors.New("invalid verification interaction")
	// ErrNotReady indicates that the verification definition is not yet available.
	ErrNotReady = errors.New("verification message is not ready")
)

// Gateway implements guild-scoped security operations.
type Gateway interface {
	// EnsureTrap repairs the unique trap channel and returns its ID.
	EnsureTrap(context.Context, string) (string, error)
	// PruneWarnings removes additional bot-authored messages from the trap.
	PruneWarnings(context.Context, string, string) error
	// GrantRole assigns a role to one member in the configured guild.
	GrantRole(context.Context, string, string) error
	// Ban bans a member without deleting their message history.
	Ban(context.Context, string) error
}

// MessageStore provides durable managed message operations.
type MessageStore interface {
	// Get reads the current desired state and remote ID.
	Get(context.Context, string) (messages.Record, error)
	// Create persists a new idempotent definition.
	Create(context.Context, messages.Definition, string) (messages.MutationResult, error)
	// Replace atomically replaces a revision.
	Replace(context.Context, string, uint64, messages.Definition, string) (messages.MutationResult, error)
	// Reconcile requests a remote integrity check.
	Reconcile(context.Context, string) error
}

// Service owns verification policy and the active trap assignment.
type Service struct {
	config   Config
	guildID  string
	messages MessageStore
	gateway  Gateway
	settings settings.Store
	trap     atomic.Value
	mutex    sync.Mutex
}

// NewService builds security policy with injected persistence and Discord.
func NewService(config Config, guildID string, store MessageStore, gateway Gateway, runtime ...settings.Store) *Service {
	service := &Service{config: config, guildID: guildID, messages: store, gateway: gateway}
	if len(runtime) > 0 {
		service.settings = runtime[0]
	}
	return service
}

// Refresh independently reconciles enabled security controls.
func (service *Service) Refresh(ctx context.Context) error {
	if !service.mutex.TryLock() {
		return nil
	}
	defer service.mutex.Unlock()
	config, err := service.Effective(ctx)
	if err != nil {
		return err
	}
	var failures []error
	if config.AntibotEnabled {
		failures = append(failures, service.refreshTrap(ctx))
	} else {
		service.trap.Store("")
	}
	if config.VerificationReady() {
		failures = append(failures, service.refreshVerification(ctx, config))
	}
	return errors.Join(failures...)
}

// Verify accepts only the currently bound managed message and grants the configured role.
func (service *Service) Verify(ctx context.Context, guildID, channelID, messageID, userID string) error {
	config, err := service.Effective(ctx)
	if err != nil {
		return err
	}
	if !config.VerificationReady() || guildID != service.guildID || channelID != config.ChannelID || !snowflakePattern.MatchString(userID) || messageID == "" {
		return ErrInvalidVerification
	}
	record, err := service.messages.Get(ctx, config.MessageKey)
	if err != nil {
		return err
	}
	if record.State == messages.StateArchived || record.GuildID != guildID || record.ChannelID != channelID || record.DiscordMessageID != messageID {
		return ErrInvalidVerification
	}
	return service.gateway.GrantRole(ctx, userID, config.RoleID)
}

// IsTrap reports whether a gateway event targets the current managed trap.
func (service *Service) IsTrap(guildID, channelID string) bool {
	active, _ := service.trap.Load().(string)
	return guildID == service.guildID && active != "" && channelID == active
}

// HandleMessage bans an author posting in the trap, excluding only this bot.
func (service *Service) HandleMessage(ctx context.Context, guildID, channelID, userID, botID string) error {
	if !service.IsTrap(guildID, channelID) || userID == botID || botID == "" || !snowflakePattern.MatchString(userID) {
		return nil
	}
	return service.gateway.Ban(ctx, userID)
}

func (service *Service) refreshTrap(ctx context.Context) error {
	record, err := service.messages.Get(ctx, TrapMessageKey)
	if err != nil && !errors.Is(err, messages.ErrNotFound) {
		return err
	}
	channelID, err := service.gateway.EnsureTrap(ctx, record.ChannelID)
	if err != nil {
		return fmt.Errorf("repair anti-bot channel: %w", err)
	}
	definition := messages.Definition{Key: TrapMessageKey, GuildID: service.guildID, ChannelID: channelID,
		Payload: messages.Payload{Components: []messages.Component{[]byte(`{"type":10,"content":"` + TrapWarning + `"}`)}}}
	if err := service.publish(ctx, record, definition); err != nil {
		return err
	}
	service.trap.Store(channelID)
	if err := service.messages.Reconcile(ctx, TrapMessageKey); err != nil {
		return err
	}
	if record.ChannelID == channelID && record.DiscordMessageID != "" {
		if err := service.gateway.PruneWarnings(ctx, channelID, record.DiscordMessageID); err != nil {
			return err
		}
	}
	return nil
}
