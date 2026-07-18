package verification

import (
	"context"
	"testing"
)

type verificationRepository struct {
	Repository
	group       Group
	memberships map[string]Membership
	deleted     bool
}

func (repository *verificationRepository) GetGroup(context.Context, string) (Group, error) {
	return repository.group, nil
}
func (repository *verificationRepository) ListGroups(context.Context, bool) ([]Group, error) {
	return []Group{repository.group}, nil
}
func (repository *verificationRepository) UpsertMembership(_ context.Context, user string, group Group) (Membership, error) {
	item := Membership{UserID: user, GroupID: group.ID, RoleID: group.RoleID}
	repository.memberships[user+group.ID] = item
	return item, nil
}
func (repository *verificationRepository) DeleteMembership(_ context.Context, user, group string) (Membership, error) {
	item, ok := repository.memberships[user+group]
	if !ok {
		return Membership{}, ErrNotFound
	}
	delete(repository.memberships, user+group)
	repository.deleted = true
	return item, nil
}
func (repository *verificationRepository) ListMemberships(_ context.Context, userID string) (Page, error) {
	page := Page{Items: []Membership{}}
	for _, membership := range repository.memberships {
		if userID == "" || membership.UserID == userID {
			page.Items = append(page.Items, membership)
		}
	}
	page.Total = len(page.Items)
	return page, nil
}

type verificationGateway struct {
	Gateway
	added, removed, dm bool
	state              MemberState
}

func (gateway *verificationGateway) MemberState(context.Context, string) (MemberState, error) {
	return gateway.state, nil
}
func (*verificationGateway) ValidateRole(context.Context, string) error { return nil }
func (gateway *verificationGateway) AddRole(context.Context, string, string) error {
	gateway.added = true
	return nil
}
func (gateway *verificationGateway) RemoveRole(context.Context, string, string) error {
	gateway.removed = true
	return nil
}
func (gateway *verificationGateway) SendVerifiedDM(context.Context, string, Group) error {
	gateway.dm = true
	return nil
}

func TestServiceSupportsIdempotentMultipleMembershipWorkflow(t *testing.T) {
	group := Group{ID: "00000000-0000-0000-0000-000000000001", Key: "member", RoleID: "123", ButtonLabel: "Member", ButtonStyle: 1, Position: 1, Enabled: true}
	repository := &verificationRepository{group: group, memberships: map[string]Membership{}}
	gateway := &verificationGateway{}
	service := NewService(repository, gateway, "456")
	if err := service.Verify(context.Background(), "456", "789", group.ID); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !gateway.added || !gateway.dm || len(repository.memberships) != 1 {
		t.Fatalf("verify state = %#v %#v", gateway, repository)
	}
	if err := service.Verify(context.Background(), "456", "789", group.ID); err != nil || len(repository.memberships) != 1 {
		t.Fatalf("repeat Verify() error = %v", err)
	}
	if err := service.Unverify(context.Background(), "789", group.ID); err != nil {
		t.Fatalf("Unverify() error = %v", err)
	}
	if !gateway.removed || !repository.deleted || len(repository.memberships) != 0 {
		t.Fatalf("unverify state = %#v %#v", gateway, repository)
	}
}

func TestServiceRejectsForeignGuild(t *testing.T) {
	service := NewService(&verificationRepository{group: Group{}, memberships: map[string]Membership{}}, &verificationGateway{}, "456")
	if err := service.Verify(context.Background(), "999", "789", "group"); err == nil {
		t.Fatal("Verify() error = nil")
	}
}
