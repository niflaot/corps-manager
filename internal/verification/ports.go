package verification

import "context"

// Repository persists verification groups and memberships.
type Repository interface {
	// CreateGroup persists one group.
	CreateGroup(context.Context, Group) (Group, error)
	// UpdateGroup replaces one group at an expected revision.
	UpdateGroup(context.Context, string, uint64, Group) (Group, error)
	// GetGroup returns one group by ID.
	GetGroup(context.Context, string) (Group, error)
	// ListGroups returns all groups in button order.
	ListGroups(context.Context, bool) ([]Group, error)
	// DeleteGroup removes an unused group at an expected revision.
	DeleteGroup(context.Context, string, uint64) error
	// UpsertMembership creates or refreshes one active membership.
	UpsertMembership(context.Context, string, Group) (Membership, error)
	// DeleteMembership hard-deletes one active membership.
	DeleteMembership(context.Context, string, string) (Membership, error)
	// ListMemberships returns filtered memberships.
	ListMemberships(context.Context, string) (Page, error)
}

// Gateway applies verification effects in the configured Discord guild.
type Gateway interface {
	// MemberState returns the user's current membership and role state.
	MemberState(context.Context, string) (MemberState, error)
	// ValidateRole verifies that a role is assignable by the bot.
	ValidateRole(context.Context, string) error
	// AddRole assigns one role to one user.
	AddRole(context.Context, string, string) error
	// RemoveRole removes one role from one user.
	RemoveRole(context.Context, string, string) error
}
