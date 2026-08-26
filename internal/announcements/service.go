package announcements

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

const actorLimit = 80

// AnnounceOpening publishes an attributed opening after atomically acquiring its cooldown.
func (service *Service) AnnounceOpening(ctx context.Context, actor string) (State, error) {
	if service.config.ChannelID == "" {
		return State{}, ErrDisabled
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = DefaultAPIActor
	}
	if utf8.RuneCountInString(actor) > actorLimit || strings.ContainsAny(actor, "\r\n") {
		return State{}, ErrInvalidActor
	}
	now := service.clock.Now().UTC()
	state, err := service.repository.Acquire(ctx, OpeningCooldownKey, now, now.Add(service.config.Cooldown), actor)
	if err != nil {
		return State{}, err
	}
	if err := service.gateway.SendOpening(ctx, service.config.ChannelID, actor); err != nil {
		if releaseErr := service.repository.Release(ctx, OpeningCooldownKey, state.AnnouncedAt); releaseErr != nil {
			return State{}, fmt.Errorf("publish opening: %w; release cooldown: %v", err, releaseErr)
		}
		return State{}, fmt.Errorf("publish opening: %w", err)
	}
	return state, nil
}

// GetCooldown returns the persisted opening cooldown.
func (service *Service) GetCooldown(ctx context.Context) (State, error) {
	return service.repository.Get(ctx, OpeningCooldownKey)
}

// ClearCooldown removes the persisted opening cooldown.
func (service *Service) ClearCooldown(ctx context.Context) error {
	return service.repository.Clear(ctx, OpeningCooldownKey)
}
