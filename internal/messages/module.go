package messages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/niflaot/corps-manager/internal/cronjob"
	"github.com/niflaot/corps-manager/platform/clock"
	"go.uber.org/fx"
)

const (
	defaultReconcileBatchSize = 25
	defaultReconcileWorkers   = 4
	reconcileJobName          = "messages-reconcile"
	reconcileJobInterval      = time.Minute
)

// Module provides managed-message services, signaling, and reconciliation.
var Module = fx.Module("messages", fx.Provide(
	NewSignal,
	fx.Annotate(provideService, fx.ParamTags("", "", `name:"guild_id"`)),
	provideReconciler,
	fx.Annotate(provideReconcileJob, fx.ResultTags(`group:"cronjobs"`)),
))

func provideService(repository Repository, signal *Signal, guildID string) *Service {
	return NewService(repository, signal, guildID)
}

func provideReconciler(repository Repository, gateway Gateway, jobClock clock.Clock, signal *Signal) *Reconciler {
	hostname, _ := os.Hostname()
	owner := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	return NewReconciler(repository, gateway, jobClock, signal, owner,
		defaultReconcileBatchSize, defaultReconcileWorkers)
}

func provideReconcileJob(reconciler *Reconciler) cronjob.Job {
	return cronjob.Job{
		Name:     reconcileJobName,
		Interval: reconcileJobInterval,
		Handler:  reconciler.ReconcileDue,
	}
}

// Service manages desired static message definitions.
type Service struct {
	repository Repository
	trigger    Trigger
	guildID    string
}

// NewService creates the managed-message application service.
func NewService(repository Repository, trigger Trigger, guildID ...string) *Service {
	configuredGuild := ""
	if len(guildID) > 0 {
		configuredGuild = guildID[0]
	}
	return &Service{repository: repository, trigger: trigger, guildID: configuredGuild}
}

// Create validates and persists one idempotent definition.
func (service *Service) Create(ctx context.Context, definition Definition, idempotencyKey string) (MutationResult, error) {
	definition.Payload = definition.Payload.Normalize()
	if service.guildID != "" && definition.GuildID != service.guildID {
		return MutationResult{}, invalid(fmt.Errorf("guildId must match the configured guild"))
	}
	if err := definition.Validate(); err != nil {
		return MutationResult{}, invalid(err)
	}
	idempotency, err := newIdempotency(idempotencyKey, "create:"+definition.Key, definition)
	if err != nil {
		return MutationResult{}, err
	}
	result, err := service.repository.Create(ctx, definition, idempotency)
	if err == nil && !result.Replay {
		service.trigger.Notify()
	}
	return result, err
}

// Get returns one managed message by logical key.
func (service *Service) Get(ctx context.Context, key string) (Record, error) {
	if !keyPattern.MatchString(key) {
		return Record{}, invalid(fmt.Errorf("invalid message key"))
	}
	return service.repository.GetByKey(ctx, key)
}

// List returns a bounded managed-message page.
func (service *Service) List(ctx context.Context, query ListQuery) (Page, error) {
	if service.guildID != "" && query.GuildID != "" && query.GuildID != service.guildID {
		return Page{}, invalid(fmt.Errorf("guildId must match the configured guild"))
	}
	if service.guildID != "" {
		query.GuildID = service.guildID
	}
	if query.Limit < 0 {
		return Page{}, invalid(fmt.Errorf("limit must not be negative"))
	}
	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		return Page{}, invalid(fmt.Errorf("offset must not be negative"))
	}
	if query.State != "" && !query.State.valid() {
		return Page{}, invalid(fmt.Errorf("invalid message state"))
	}
	return service.repository.List(ctx, query)
}

// Replace atomically replaces desired state at the expected revision.
func (service *Service) Replace(ctx context.Context, key string, revision uint64, definition Definition, idempotencyKey string) (MutationResult, error) {
	if definition.Key != "" && definition.Key != key {
		return MutationResult{}, invalid(fmt.Errorf("message key is immutable"))
	}
	definition.Key = key
	definition.Payload = definition.Payload.Normalize()
	if service.guildID != "" && definition.GuildID != service.guildID {
		return MutationResult{}, invalid(fmt.Errorf("guildId must match the configured guild"))
	}
	if revision == 0 {
		return MutationResult{}, invalid(fmt.Errorf("revision is required"))
	}
	if err := definition.Validate(); err != nil {
		return MutationResult{}, invalid(err)
	}
	idempotency, err := newIdempotency(idempotencyKey, "replace:"+key, struct {
		// Revision is the requested optimistic concurrency version.
		Revision uint64
		// Definition is the requested desired state.
		Definition Definition
	}{revision, definition})
	if err != nil {
		return MutationResult{}, err
	}
	result, err := service.repository.Replace(ctx, key, revision, definition, idempotency)
	if err == nil && !result.Replay {
		service.trigger.Notify()
	}
	return result, err
}

// Archive stops managing one message at the expected revision.
func (service *Service) Archive(ctx context.Context, key string, revision uint64, idempotencyKey string) (MutationResult, error) {
	if revision == 0 || !keyPattern.MatchString(key) {
		return MutationResult{}, invalid(fmt.Errorf("valid key and revision are required"))
	}
	idempotency, err := newIdempotency(idempotencyKey, "archive:"+key, revision)
	if err != nil {
		return MutationResult{}, err
	}
	return service.repository.Archive(ctx, key, revision, idempotency)
}

// Reconcile marks one definition due and wakes the worker.
func (service *Service) Reconcile(ctx context.Context, key string) error {
	if !keyPattern.MatchString(key) {
		return invalid(fmt.Errorf("invalid message key"))
	}
	if err := service.repository.MarkDue(ctx, key); err != nil {
		return err
	}
	service.trigger.Notify()
	return nil
}

// Signal is a coalescing in-process reconciliation trigger.
type Signal struct{ channel chan struct{} }

// NewSignal creates a non-blocking reconciliation signal.
func NewSignal() *Signal { return &Signal{channel: make(chan struct{}, 1)} }

// Notify coalesces a pending reconciliation wakeup.
func (signal *Signal) Notify() {
	select {
	case signal.channel <- struct{}{}:
	default:
	}
}

// C returns the reconciliation wakeup channel.
func (signal *Signal) C() <-chan struct{} { return signal.channel }

func newIdempotency(key string, operation string, request any) (Idempotency, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 128 {
		return Idempotency{}, invalid(fmt.Errorf("Idempotency-Key is required and must not exceed 128 characters"))
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return Idempotency{}, fmt.Errorf("encode idempotency request: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return Idempotency{Key: key, Operation: operation, RequestHash: hex.EncodeToString(digest[:])}, nil
}

func invalid(err error) error { return fmt.Errorf("%w: %v", ErrInvalidDefinition, err) }
