package verification

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	membershipReconcileWorkers = 4
	memberLockStripes          = 64
	fnvOffset32                = uint32(2166136261)
	fnvPrime32                 = uint32(16777619)
)

type memberLocks [memberLockStripes]sync.Mutex

func (locks *memberLocks) lock(userID string) func() {
	hash := fnvOffset32
	for index := 0; index < len(userID); index++ {
		hash ^= uint32(userID[index])
		hash *= fnvPrime32
	}
	mutex := &locks[hash%memberLockStripes]
	mutex.Lock()
	return mutex.Unlock
}

// ReconcileMember repairs or removes one user's persisted verification state.
func (service *Service) ReconcileMember(ctx context.Context, guildID, userID string) error {
	if guildID != service.guildID || !snowflakePattern.MatchString(userID) {
		return fmt.Errorf("%w: member outside configured guild", ErrInvalid)
	}
	unlock := service.locks.lock(userID)
	defer unlock()
	return service.reconcileMember(ctx, userID)
}

// ReconcileMemberships repairs every persisted verification membership.
func (service *Service) ReconcileMemberships(ctx context.Context) error {
	page, err := service.repository.ListMemberships(ctx, "")
	if err != nil {
		return err
	}
	users := make(map[string]struct{}, len(page.Items))
	for _, membership := range page.Items {
		users[membership.UserID] = struct{}{}
	}
	var workers errgroup.Group
	workers.SetLimit(membershipReconcileWorkers)
	for userID := range users {
		userID := userID
		workers.Go(func() error {
			return service.ReconcileMember(ctx, service.guildID, userID)
		})
	}
	return workers.Wait()
}

func (service *Service) reconcileMember(ctx context.Context, userID string) error {
	page, err := service.repository.ListMemberships(ctx, userID)
	if err != nil || len(page.Items) == 0 {
		return err
	}
	state, err := service.gateway.MemberState(ctx, userID)
	if err != nil {
		return err
	}
	for _, membership := range page.Items {
		if err := service.reconcileMembership(ctx, state, membership); err != nil {
			return err
		}
	}
	return nil
}

func (service *Service) reconcileMembership(ctx context.Context, state MemberState, membership Membership) error {
	stale := !state.Present || (!state.JoinedAt.IsZero() && membership.VerifiedAt.Before(state.JoinedAt))
	if stale {
		if state.HasRole(membership.RoleID) {
			if err := service.gateway.RemoveRole(ctx, membership.UserID, membership.RoleID); err != nil {
				return err
			}
		}
		_, err := service.repository.DeleteMembership(ctx, membership.UserID, membership.GroupID)
		if err == ErrNotFound {
			return nil
		}
		return err
	}
	if !state.HasRole(membership.RoleID) {
		return service.gateway.AddRole(ctx, membership.UserID, membership.RoleID)
	}
	return nil
}
