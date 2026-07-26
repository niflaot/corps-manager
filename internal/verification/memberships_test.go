package verification

import (
	"context"
	"testing"
	"time"

	"github.com/pixelados-net/discord-bot/internal/verification/notification"
)

const (
	testGuildID = "456"
	testUserID  = "789"
	testGroupID = "00000000-0000-0000-0000-000000000001"
	testRoleID  = "123"
)

func TestReconcileMemberDeletesDepartedMembership(t *testing.T) {
	membership := testMembership(time.Now().Add(-time.Hour))
	repository := testMembershipRepository(membership)
	service := testService(repository, &verificationGateway{state: MemberState{Present: false}})
	if err := service.ReconcileMember(context.Background(), testGuildID, testUserID); err != nil {
		t.Fatalf("ReconcileMember() error = %v", err)
	}
	if len(repository.memberships) != 0 {
		t.Fatalf("memberships = %#v", repository.memberships)
	}
}

func TestReconcileMemberInvalidatesMembershipFromPreviousJoin(t *testing.T) {
	now := time.Now()
	membership := testMembership(now.Add(-time.Hour))
	repository := testMembershipRepository(membership)
	gateway := &verificationGateway{state: MemberState{
		Present: true, JoinedAt: now, RoleIDs: map[string]struct{}{testRoleID: {}},
	}}
	service := testService(repository, gateway)
	if err := service.ReconcileMember(context.Background(), testGuildID, testUserID); err != nil {
		t.Fatalf("ReconcileMember() error = %v", err)
	}
	if len(repository.memberships) != 0 || !gateway.removed {
		t.Fatalf("reconciled state = %#v %#v", repository, gateway)
	}
}

func TestReconcileMemberRestoresCurrentMembershipRole(t *testing.T) {
	now := time.Now()
	membership := testMembership(now)
	repository := testMembershipRepository(membership)
	gateway := &verificationGateway{state: MemberState{Present: true, JoinedAt: now.Add(-time.Hour), RoleIDs: map[string]struct{}{}}}
	service := testService(repository, gateway)
	if err := service.ReconcileMember(context.Background(), testGuildID, testUserID); err != nil {
		t.Fatalf("ReconcileMember() error = %v", err)
	}
	if len(repository.memberships) != 1 || !gateway.added {
		t.Fatalf("reconciled state = %#v %#v", repository, gateway)
	}
}

func testMembership(verifiedAt time.Time) Membership {
	return Membership{ID: "membership", UserID: testUserID, GroupID: testGroupID, RoleID: testRoleID, VerifiedAt: verifiedAt}
}

func testMembershipRepository(membership Membership) *verificationRepository {
	return &verificationRepository{
		group:       Group{ID: testGroupID, RoleID: testRoleID},
		memberships: map[string]Membership{testUserID + testGroupID: membership},
	}
}

func testService(repository Repository, gateway Gateway) *Service {
	return NewService(repository, gateway, &notificationPublisher{events: map[string]notification.Event{}}, testGuildID)
}
