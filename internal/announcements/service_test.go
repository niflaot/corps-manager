package announcements

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/niflaot/corps-manager/platform/clock"
)

type memoryRepository struct{ state *State }

func (repository *memoryRepository) Acquire(_ context.Context, key string, announcedAt time.Time,
	availableAt time.Time, actor string) (State, error) {
	if repository.state != nil && repository.state.AvailableAt.After(announcedAt) {
		return State{}, &CooldownActiveError{State: *repository.state}
	}
	state := State{Key: key, Actor: actor, AnnouncedAt: announcedAt, AvailableAt: availableAt}
	repository.state = &state
	return state, nil
}

func (repository *memoryRepository) Get(_ context.Context, _ string) (State, error) {
	if repository.state == nil {
		return State{}, ErrNotFound
	}
	return *repository.state, nil
}

func (repository *memoryRepository) Release(_ context.Context, _ string, announcedAt time.Time) error {
	if repository.state != nil && repository.state.AnnouncedAt.Equal(announcedAt) {
		repository.state = nil
	}
	return nil
}

func (repository *memoryRepository) Clear(context.Context, string) error {
	repository.state = nil
	return nil
}

type gatewayStub struct {
	err    error
	actors []string
}

func (gateway *gatewayStub) SendOpening(_ context.Context, _ string, actor string) error {
	gateway.actors = append(gateway.actors, actor)
	return gateway.err
}

func TestServiceEnforcesAndClearsCooldown(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository := &memoryRepository{}
	gateway := &gatewayStub{}
	fakeClock := clock.NewFake(now)
	service := NewService(Config{ChannelID: "123", Cooldown: 30 * time.Minute}, repository, gateway, fakeClock)

	state, err := service.AnnounceOpening(context.Background(), " Thomas J. ")
	if err != nil || state.Actor != "Thomas J." || !state.AvailableAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("first announcement = %#v, %v", state, err)
	}
	if _, err := service.AnnounceOpening(context.Background(), "Thomas J."); !errors.Is(err, ErrCooldownActive) {
		t.Fatalf("second announcement error = %v", err)
	}
	if err := service.ClearCooldown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AnnounceOpening(context.Background(), ""); err != nil {
		t.Fatalf("announcement after clear: %v", err)
	}
	if gateway.actors[len(gateway.actors)-1] != DefaultAPIActor {
		t.Fatalf("default actor = %q", gateway.actors[len(gateway.actors)-1])
	}
}

func TestServiceReleasesCooldownWhenPublicationFails(t *testing.T) {
	repository := &memoryRepository{}
	gateway := &gatewayStub{err: errors.New("discord offline")}
	service := NewService(Config{ChannelID: "123", Cooldown: time.Minute}, repository, gateway,
		clock.NewFake(time.Now()))
	if _, err := service.AnnounceOpening(context.Background(), "Thomas J."); err == nil {
		t.Fatal("expected publication error")
	}
	if repository.state != nil {
		t.Fatalf("failed publication retained cooldown: %#v", repository.state)
	}
}

func TestServiceRejectsInvalidActorAndDisabledChannel(t *testing.T) {
	repository := &memoryRepository{}
	gateway := &gatewayStub{}
	service := NewService(Config{Cooldown: time.Minute}, repository, gateway, clock.NewFake(time.Now()))
	if _, err := service.AnnounceOpening(context.Background(), "actor"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled error = %v", err)
	}
	service.config.ChannelID = "123"
	if _, err := service.AnnounceOpening(context.Background(), "bad\nactor"); !errors.Is(err, ErrInvalidActor) {
		t.Fatalf("invalid actor error = %v", err)
	}
}
