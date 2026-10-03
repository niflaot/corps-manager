package security

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/niflaot/corps-manager/internal/messages"
)

type messageStub struct {
	records   map[string]messages.Record
	mutations int
	err       error
}

func (store *messageStub) Get(_ context.Context, key string) (messages.Record, error) {
	if store.err != nil {
		return messages.Record{}, store.err
	}
	item, found := store.records[key]
	if !found {
		return messages.Record{}, messages.ErrNotFound
	}
	return item, nil
}
func (store *messageStub) Create(_ context.Context, definition messages.Definition, _ string) (messages.MutationResult, error) {
	return store.save(definition, 1)
}
func (store *messageStub) Replace(_ context.Context, _ string, revision uint64, definition messages.Definition, _ string) (messages.MutationResult, error) {
	return store.save(definition, revision+1)
}
func (store *messageStub) save(definition messages.Definition, revision uint64) (messages.MutationResult, error) {
	if err := definition.Validate(); err != nil {
		return messages.MutationResult{}, err
	}
	hash, _ := definition.Payload.Hash()
	record := messages.Record{Definition: definition, Revision: revision, DesiredHash: hash, State: messages.StatePending}
	store.records[definition.Key] = record
	store.mutations++
	return messages.MutationResult{Record: record}, nil
}
func (*messageStub) Reconcile(context.Context, string) error { return nil }

type gatewayStub struct {
	trapID    string
	preferred string
	grants    int
	bans      []string
	err       error
}

func (gateway *gatewayStub) EnsureTrap(_ context.Context, preferred string) (string, error) {
	gateway.preferred = preferred
	return gateway.trapID, gateway.err
}
func (*gatewayStub) PruneWarnings(context.Context, string, string) error { return nil }
func (gateway *gatewayStub) GrantRole(context.Context, string, string) error {
	gateway.grants++
	return gateway.err
}
func (gateway *gatewayStub) Ban(_ context.Context, id string) error {
	gateway.bans = append(gateway.bans, id)
	return gateway.err
}

func verificationRecord() messages.Record {
	return messages.Record{Definition: messages.Definition{Key: "verification", GuildID: "123", ChannelID: "456",
		Payload: messages.Payload{Components: []messages.Component{[]byte(`{"type":10,"content":"Mis reglas"}`)}}},
		DiscordMessageID: "789", State: messages.StateHealthy, Revision: 1}
}

func TestRefreshPreservesContentAndIsIdempotent(t *testing.T) {
	store := &messageStub{records: map[string]messages.Record{"verification": verificationRecord()}}
	gateway := &gatewayStub{trapID: "654"}
	service := NewService(Config{VerificationEnabled: true, ChannelID: "456", RoleID: "321", MessageKey: "verification", AntibotEnabled: true}, "123", store, gateway)
	for range 3 {
		if err := service.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if store.mutations != 2 {
		t.Fatalf("mutations = %d; expected one for each control", store.mutations)
	}
	record := store.records["verification"]
	if !strings.Contains(string(record.Payload.Components[0]), "Mis reglas") || len(record.Payload.Components) != 2 {
		t.Fatalf("user payload changed: %#v", record.Payload)
	}
	if gateway.preferred != "654" || !service.IsTrap("123", "654") {
		t.Fatal("trap identity was not retained")
	}
}

func TestVerificationRejectsForeignAndStaleInteractions(t *testing.T) {
	store := &messageStub{records: map[string]messages.Record{"verification": verificationRecord()}}
	gateway := &gatewayStub{}
	service := NewService(Config{VerificationEnabled: true, ChannelID: "456", RoleID: "321", MessageKey: "verification"}, "123", store, gateway)
	for _, binding := range [][3]string{{"999", "456", "789"}, {"123", "999", "789"}, {"123", "456", "999"}, {"123", "456", ""}} {
		if err := service.Verify(context.Background(), binding[0], binding[1], binding[2], "111"); !errors.Is(err, ErrInvalidVerification) {
			t.Fatalf("foreign binding accepted: %v", err)
		}
	}
	if gateway.grants != 0 {
		t.Fatal("foreign interaction granted a role")
	}
	for range 2 {
		if err := service.Verify(context.Background(), "123", "456", "789", "111"); err != nil {
			t.Fatal(err)
		}
	}
	if gateway.grants != 2 {
		t.Fatal("valid verification was not delegated to idempotent role endpoint")
	}
	record := store.records["verification"]
	record.State = messages.StateArchived
	store.records["verification"] = record
	if err := service.Verify(context.Background(), "123", "456", "789", "111"); !errors.Is(err, ErrInvalidVerification) {
		t.Fatal(err)
	}
}

func TestTrapBansOnlyMatchingGuildChannelAndNeverSelf(t *testing.T) {
	gateway := &gatewayStub{trapID: "654"}
	store := &messageStub{records: map[string]messages.Record{}}
	service := NewService(Config{AntibotEnabled: true}, "123", store, gateway)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, event := range [][3]string{{"999", "654", "111"}, {"123", "456", "111"}, {"123", "654", "999"}, {"123", "654", "111"}} {
		if err := service.HandleMessage(context.Background(), event[0], event[1], event[2], "999"); err != nil {
			t.Fatal(err)
		}
	}
	if len(gateway.bans) != 1 || gateway.bans[0] != "111" {
		t.Fatalf("bans = %v", gateway.bans)
	}
}

func TestMissingVerificationDoesNotPreventTrapRepair(t *testing.T) {
	store := &messageStub{records: map[string]messages.Record{}}
	gateway := &gatewayStub{trapID: "654"}
	service := NewService(Config{AntibotEnabled: true, VerificationEnabled: true, MessageKey: "missing"}, "123", store, gateway)
	if err := service.Refresh(context.Background()); !errors.Is(err, messages.ErrNotFound) {
		t.Fatal(err)
	}
	if !service.IsTrap("123", "654") {
		t.Fatal("missing verification disabled trap")
	}
}

func TestFailedChannelRepairDoesNotArmArbitraryChannel(t *testing.T) {
	gateway := &gatewayStub{trapID: "654", err: errors.New("permission denied")}
	service := NewService(Config{AntibotEnabled: true}, "123", &messageStub{records: map[string]messages.Record{}}, gateway)
	if err := service.Refresh(context.Background()); err == nil {
		t.Fatal("expected channel error")
	}
	if service.IsTrap("123", "654") {
		t.Fatal("failed repair armed trap")
	}
}
