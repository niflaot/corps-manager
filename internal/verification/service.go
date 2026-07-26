package verification

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pixelados-net/discord-bot/internal/verification/notification"
)

// Service manages verification configuration and idempotent memberships.
type Service struct {
	repository    Repository
	gateway       Gateway
	notifications notification.Publisher
	guildID       string
	locks         memberLocks
}

// NewService creates a guild-scoped verification service.
func NewService(repository Repository, gateway Gateway, notifications notification.Publisher, guildID string) *Service {
	return &Service{repository: repository, gateway: gateway, notifications: notifications, guildID: guildID}
}

// CreateGroup validates Discord limits and creates one verification group.
func (service *Service) CreateGroup(ctx context.Context, group Group) (Group, error) {
	group.ID = ""
	if err := service.validateGroup(ctx, group); err != nil {
		return Group{}, err
	}
	groups, err := service.repository.ListGroups(ctx, false)
	if err != nil {
		return Group{}, err
	}
	if len(groups) >= MaximumGroups {
		return Group{}, fmt.Errorf("%w: at most %d groups", ErrInvalid, MaximumGroups)
	}
	return service.repository.CreateGroup(ctx, group)
}

// UpdateGroup replaces one verification group.
func (service *Service) UpdateGroup(ctx context.Context, id string, revision uint64, group Group) (Group, error) {
	if uuid.Validate(id) != nil {
		return Group{}, fmt.Errorf("%w: invalid group ID", ErrInvalid)
	}
	if revision == 0 {
		return Group{}, fmt.Errorf("%w: revision is required", ErrInvalid)
	}
	group.ID = id
	if err := service.validateGroup(ctx, group); err != nil {
		return Group{}, err
	}
	return service.repository.UpdateGroup(ctx, id, revision, group)
}

// GetGroup returns one verification group.
func (service *Service) GetGroup(ctx context.Context, id string) (Group, error) {
	if uuid.Validate(id) != nil {
		return Group{}, fmt.Errorf("%w: invalid group ID", ErrInvalid)
	}
	return service.repository.GetGroup(ctx, id)
}

// ListGroups returns verification groups in button order.
func (service *Service) ListGroups(ctx context.Context, enabledOnly bool) ([]Group, error) {
	return service.repository.ListGroups(ctx, enabledOnly)
}

// DeleteGroup removes one group without active memberships.
func (service *Service) DeleteGroup(ctx context.Context, id string, revision uint64) error {
	if uuid.Validate(id) != nil {
		return fmt.Errorf("%w: invalid group ID", ErrInvalid)
	}
	if revision == 0 {
		return fmt.Errorf("%w: revision is required", ErrInvalid)
	}
	return service.repository.DeleteGroup(ctx, id, revision)
}

// ListMemberships returns memberships optionally filtered by user.
func (service *Service) ListMemberships(ctx context.Context, userID string) (Page, error) {
	if userID != "" && !snowflakePattern.MatchString(userID) {
		return Page{}, fmt.Errorf("%w: invalid userId", ErrInvalid)
	}
	return service.repository.ListMemberships(ctx, userID)
}

// Verify idempotently adds a group's role and membership, then attempts its DM.
func (service *Service) Verify(ctx context.Context, guildID, userID, groupID string) error {
	if guildID != service.guildID || !snowflakePattern.MatchString(userID) || uuid.Validate(groupID) != nil {
		return fmt.Errorf("%w: interaction outside configured guild", ErrInvalid)
	}
	unlock := service.locks.lock(userID)
	defer unlock()
	group, err := service.repository.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if !group.Enabled {
		return fmt.Errorf("%w: group is disabled", ErrInvalid)
	}
	if err := service.gateway.AddRole(ctx, userID, group.RoleID); err != nil {
		return err
	}
	membership, err := service.repository.UpsertMembership(ctx, userID, group)
	if err != nil {
		_ = service.gateway.RemoveRole(ctx, userID, group.RoleID)
		return err
	}
	service.notifications.Enqueue(ctx, notification.NewEvent(
		notification.KindVerified, membership.ID, userID, group.ID, group.Key,
	))
	return nil
}

// Unverify removes one role and hard-deletes its membership.
func (service *Service) Unverify(ctx context.Context, userID, groupID string) error {
	if !snowflakePattern.MatchString(userID) || uuid.Validate(groupID) != nil {
		return fmt.Errorf("%w: invalid user", ErrInvalid)
	}
	unlock := service.locks.lock(userID)
	defer unlock()
	group, err := service.repository.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if err := service.gateway.RemoveRole(ctx, userID, group.RoleID); err != nil {
		return err
	}
	membership, err := service.repository.DeleteMembership(ctx, userID, groupID)
	if err == ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	service.notifications.Enqueue(ctx, notification.NewEvent(
		notification.KindUnverified, membership.ID, userID, group.ID, group.Key,
	))
	return nil
}

func (service *Service) validateGroup(ctx context.Context, group Group) error {
	if !groupKeyPattern.MatchString(group.Key) || !snowflakePattern.MatchString(group.RoleID) {
		return fmt.Errorf("%w: invalid key or roleId", ErrInvalid)
	}
	if strings.TrimSpace(group.ButtonLabel) == "" || utf8.RuneCountInString(group.ButtonLabel) > 80 || len(group.ButtonEmoji) > 32 {
		return fmt.Errorf("%w: invalid button label or emoji", ErrInvalid)
	}
	if group.ButtonStyle < 1 || group.ButtonStyle > 4 || group.Position < 1 || group.Position > MaximumGroups {
		return fmt.Errorf("%w: style must be 1-4 and position 1-5", ErrInvalid)
	}
	if err := service.gateway.ValidateRole(ctx, group.RoleID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return nil
}
