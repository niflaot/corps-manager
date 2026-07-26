package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/zap"
)

type dispatcherRepository struct {
	Repository
	// deliveries contains the next claimed batch.
	deliveries []Delivery
	// completions records successful delivery updates.
	completions []Completion
	// releases records retry and dead-letter updates.
	releases []Release
	// claimErr is the configured claim failure.
	claimErr error
	// completeErr is the configured completion failure.
	completeErr error
	// releaseErr is the configured release failure.
	releaseErr error
}

// ClaimDue returns the configured test batch once.
func (repository *dispatcherRepository) ClaimDue(context.Context, ClaimRequest) ([]Delivery, error) {
	deliveries := repository.deliveries
	repository.deliveries = nil
	return deliveries, repository.claimErr
}

// Complete records one test completion.
func (repository *dispatcherRepository) Complete(_ context.Context, completion Completion) error {
	repository.completions = append(repository.completions, completion)
	return repository.completeErr
}

// Release records one test failure transition.
func (repository *dispatcherRepository) Release(_ context.Context, release Release) error {
	repository.releases = append(repository.releases, release)
	return repository.releaseErr
}

type dispatcherSender struct {
	// messageID is the successful delivery result.
	messageID string
	// err is the configured Discord failure.
	err error
}

// Send returns the configured test delivery result.
func (sender *dispatcherSender) Send(context.Context, Delivery) (string, error) {
	return sender.messageID, sender.err
}

// TestDispatcherCompletesSuccessfulDelivery verifies the happy path.
func TestDispatcherCompletesSuccessfulDelivery(t *testing.T) {
	repository := &dispatcherRepository{deliveries: []Delivery{{ID: "notification"}}}
	dispatcher := testDispatcher(repository, &dispatcherSender{messageID: "message"}, 3)
	if err := dispatcher.DispatchDue(context.Background()); err != nil {
		t.Fatalf("DispatchDue() error = %v", err)
	}
	if len(repository.completions) != 1 || repository.completions[0].DiscordMessageID != "message" ||
		len(repository.releases) != 0 {
		t.Fatalf("delivery state = %#v %#v", repository.completions, repository.releases)
	}
}

// TestDispatcherRetriesThenMovesDeliveryToDeadLetter verifies bounded failure handling.
func TestDispatcherRetriesThenMovesDeliveryToDeadLetter(t *testing.T) {
	repository := &dispatcherRepository{deliveries: []Delivery{{ID: "retry", Attempts: 0}}}
	sender := &dispatcherSender{err: errors.New("closed direct messages")}
	dispatcher := testDispatcher(repository, sender, 2)
	if err := dispatcher.DispatchDue(context.Background()); err != nil {
		t.Fatalf("first DispatchDue() error = %v", err)
	}
	if len(repository.releases) != 1 || repository.releases[0].State != StateRetry {
		t.Fatalf("first release = %#v", repository.releases)
	}
	repository.deliveries = []Delivery{{ID: "retry", Attempts: 1}}
	if err := dispatcher.DispatchDue(context.Background()); err != nil {
		t.Fatalf("second DispatchDue() error = %v", err)
	}
	if len(repository.releases) != 2 || repository.releases[1].State != StateDead {
		t.Fatalf("second release = %#v", repository.releases)
	}
}

// TestDispatcherPropagatesRepositoryFailures verifies lease state errors remain visible.
func TestDispatcherPropagatesRepositoryFailures(t *testing.T) {
	expected := errors.New("repository unavailable")
	repository := &dispatcherRepository{claimErr: expected}
	dispatcher := testDispatcher(repository, &dispatcherSender{}, 2)
	if err := dispatcher.DispatchDue(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("claim error = %v", err)
	}
	repository.claimErr = nil
	repository.completeErr = expected
	repository.deliveries = []Delivery{{ID: "complete"}}
	dispatcher.sender = &dispatcherSender{messageID: "message"}
	if err := dispatcher.DispatchDue(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("complete error = %v", err)
	}
	repository.completeErr = nil
	repository.releaseErr = expected
	repository.deliveries = []Delivery{{ID: "release"}}
	dispatcher.sender = &dispatcherSender{err: errors.New("Discord unavailable")}
	if err := dispatcher.DispatchDue(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("release error = %v", err)
	}
}

// TestDispatcherRunStopsWithContext verifies context-bound worker shutdown.
func TestDispatcherRunStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dispatcher := testDispatcher(&dispatcherRepository{}, &dispatcherSender{}, 2)
	if err := dispatcher.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

// TestLoadConfigAndModuleProviders verifies default worker wiring.
func TestLoadConfigAndModuleProviders(t *testing.T) {
	for _, key := range []string{
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_INTERVAL",
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_BATCH_SIZE",
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_WORKERS",
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_LEASE_DURATION",
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_MAX_ATTEMPTS",
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_RETRY_BASE",
		"DISCORD_BOT_VERIFICATION_NOTIFICATIONS_RETRY_MAX",
	} {
		t.Setenv(key, "")
	}
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	repository := &dispatcherRepository{}
	signal := NewSignal()
	log := zap.NewNop()
	service := provideService(repository, signal, log)
	dispatcher := provideDispatcher(repository, &dispatcherSender{}, clock.NewFake(time.Unix(100, 0)), signal, log, config)
	job := provideDispatchJob(dispatcher, config)
	if service == nil || dispatcher == nil || job.Name != dispatchJobName || job.Interval != config.Interval {
		t.Fatalf("providers = %#v %#v %#v", service, dispatcher, job)
	}
}

// TestLoadConfigRejectsInvalidRetryPolicy verifies startup validation.
func TestLoadConfigRejectsInvalidRetryPolicy(t *testing.T) {
	t.Setenv("DISCORD_BOT_VERIFICATION_NOTIFICATIONS_MAX_ATTEMPTS", "0")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() error = nil")
	}
}

// testDispatcher creates a deterministic test worker.
func testDispatcher(repository Repository, sender Sender, attempts int) *Dispatcher {
	config := Config{
		BatchSize: 10, Workers: 2, LeaseDuration: time.Minute, MaxAttempts: attempts,
		RetryBase: time.Second, RetryMax: time.Minute,
	}
	return NewDispatcher(repository, sender, clock.NewFake(time.Unix(100, 0)), NewSignal(), zap.NewNop(), config, "worker")
}
