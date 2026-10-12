package messagesdiscord

import (
	"testing"
	"time"

	"github.com/niflaot/corps-manager/platform/clock"
)

func TestPendingStoreExpiresBindsUserAndConsumesOnce(t *testing.T) {
	fake := clock.NewFake(time.Unix(1000, 0))
	store := newPendingStore(fake)
	id, err := store.put(pending{Key: "a", UserID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, result := store.take(id, "2"); result != takeForeign {
		t.Fatal("another user took the preview")
	}
	if item, result := store.take(id, "1"); result != takeOK || item.Key != "a" {
		t.Fatal("owner could not confirm")
	}
	if _, result := store.take(id, "1"); result != takeMissing {
		t.Fatal("confirmation replayed")
	}
	id, _ = store.put(pending{UserID: "1"})
	fake.Advance(pendingTTL + time.Second)
	if _, result := store.take(id, "1"); result != takeMissing {
		t.Fatal("expired preview accepted")
	}
}

func TestPendingStoreIsBounded(t *testing.T) {
	store := newPendingStore(clock.NewFake(time.Unix(1000, 0)))
	for range pendingLimit {
		if _, err := store.put(pending{UserID: "1"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.put(pending{UserID: "1"}); err == nil {
		t.Fatal("store grew past its limit")
	}
}
