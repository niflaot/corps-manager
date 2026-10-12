package messagesdiscord

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/niflaot/corps-manager/internal/messages"
	"github.com/niflaot/corps-manager/platform/clock"
)

const (
	pendingTTL = 15 * time.Minute
	// pendingLimit bounds unconfirmed previews held in memory.
	pendingLimit = 128
)

var errPendingFull = errors.New("too many pending previews")

type action string

const (
	actionCreate  action = "c"
	actionEdit    action = "e"
	actionPreview action = "p"
	actionArchive action = "a"
)

type pending struct {
	Action    action
	Key       string
	ChannelID string
	Payload   messages.Payload
	UserID    string
	Expires   time.Time
}

type takeResult int

const (
	takeOK takeResult = iota
	takeMissing
	takeForeign
)

type pendingStore struct {
	mutex sync.Mutex
	items map[string]pending
	clock clock.Clock
}

func newPendingStore(source clock.Clock) *pendingStore {
	return &pendingStore{items: map[string]pending{}, clock: source}
}

func (store *pendingStore) put(item pending) (string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	now := store.clock.Now()
	for id, existing := range store.items {
		if !existing.Expires.After(now) {
			delete(store.items, id)
		}
	}
	if len(store.items) >= pendingLimit {
		return "", errPendingFull
	}
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buffer)
	item.Expires = now.Add(pendingTTL)
	store.items[id] = item
	return id, nil
}

func (store *pendingStore) take(id, userID string) (pending, takeResult) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	item, found := store.items[id]
	if !found || !item.Expires.After(store.clock.Now()) {
		delete(store.items, id)
		return pending{}, takeMissing
	}
	if item.UserID != userID {
		return pending{}, takeForeign
	}
	delete(store.items, id)
	return item, takeOK
}

func (store *pendingStore) discard(id, userID string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if item, found := store.items[id]; found && item.UserID == userID {
		delete(store.items, id)
	}
}
