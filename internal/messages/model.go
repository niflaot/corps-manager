// Package messages manages durable static Discord messages.
package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"time"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

// State identifies reconciliation state.
type State string

const (
	// StatePending awaits reconciliation.
	StatePending State = "pending"
	// StateHealthy matches the desired definition.
	StateHealthy State = "healthy"
	// StateDrifted differs from the desired definition.
	StateDrifted State = "drifted"
	// StateRepairing is leased by a reconciler.
	StateRepairing State = "repairing"
	// StateBlocked requires intervention or delayed retry.
	StateBlocked State = "blocked"
	// StateArchived is no longer managed.
	StateArchived State = "archived"
)

// Component contains one canonical Discord Components V2 object.
type Component json.RawMessage

// MarshalJSON preserves the component's JSON object representation.
func (component Component) MarshalJSON() ([]byte, error) {
	if !json.Valid(component) {
		return nil, fmt.Errorf("invalid component JSON")
	}
	return component, nil
}

// UnmarshalJSON stores one component JSON object without base64 conversion.
func (component *Component) UnmarshalJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid component JSON")
	}
	*component = append((*component)[:0], data...)
	return nil
}

// AllowedMentions controls which mentions Discord may notify.
type AllowedMentions struct {
	// Parse contains Discord mention categories.
	Parse []string `json:"parse"`
	// Roles contains explicitly allowed role snowflakes.
	Roles []string `json:"roles,omitempty"`
	// Users contains explicitly allowed user snowflakes.
	Users []string `json:"users,omitempty"`
	// RepliedUser controls notification of a replied user.
	RepliedUser bool `json:"repliedUser,omitempty"`
}

// Payload contains a managed Discord Components V2 message body.
type Payload struct {
	// Components contains the top-level Components V2 layout.
	Components []Component `json:"components"`
	// AllowedMentions prevents accidental notifications.
	AllowedMentions AllowedMentions `json:"allowedMentions"`
}

// Definition is the desired static message and channel assignment.
type Definition struct {
	// Key is the immutable logical identifier.
	Key string `json:"key"`
	// GuildID is the assigned Discord guild snowflake.
	GuildID string `json:"guildId"`
	// ChannelID is the assigned Discord channel snowflake.
	ChannelID string `json:"channelId"`
	// Payload is the desired message body.
	Payload Payload `json:"payload"`
}

// Record is a persisted managed message.
type Record struct {
	Definition
	// ID is the internal UUID.
	ID string `json:"id"`
	// DiscordMessageID is the current remote message snowflake.
	DiscordMessageID string `json:"discordMessageId,omitempty"`
	// DesiredHash identifies the canonical desired payload.
	DesiredHash string `json:"desiredHash"`
	// ObservedHash identifies the last observed remote payload.
	ObservedHash string `json:"observedHash,omitempty"`
	// Revision is the optimistic concurrency version.
	Revision uint64 `json:"revision"`
	// State is the reconciliation state.
	State State `json:"state"`
	// LastCheckedAt is the last remote observation.
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	// LastRepairedAt is the last successful mutation.
	LastRepairedAt *time.Time `json:"lastRepairedAt,omitempty"`
	// LastError is a sanitized operational error.
	LastError string `json:"lastError,omitempty"`
	// FailureCount is the consecutive reconciliation failure count.
	FailureCount int `json:"failureCount"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the last desired-state update.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize returns a stable payload representation.
func (payload Payload) Normalize() Payload {
	payload.Components = slices.Clone(payload.Components)
	if payload.Components == nil {
		payload.Components = []Component{}
	}
	for index, component := range payload.Components {
		var value map[string]any
		if json.Unmarshal(component, &value) == nil {
			normalizeComponentValue(value)
			if encoded, err := json.Marshal(value); err == nil {
				payload.Components[index] = Component(encoded)
			}
		}
	}
	payload.AllowedMentions.Parse = slices.Clone(payload.AllowedMentions.Parse)
	payload.AllowedMentions.Roles = slices.Clone(payload.AllowedMentions.Roles)
	payload.AllowedMentions.Users = slices.Clone(payload.AllowedMentions.Users)
	slices.Sort(payload.AllowedMentions.Parse)
	slices.Sort(payload.AllowedMentions.Roles)
	slices.Sort(payload.AllowedMentions.Users)
	if payload.AllowedMentions.Parse == nil {
		payload.AllowedMentions.Parse = []string{}
	}
	return payload
}

func normalizeComponentValue(component map[string]any) {
	delete(component, "id")
	deleteFalseComponentField(component, "disabled")
	deleteFalseComponentField(component, "spoiler")
	if children, ok := component["components"].([]any); ok {
		for _, child := range children {
			if object, objectOK := child.(map[string]any); objectOK {
				normalizeComponentValue(object)
			}
		}
	}
	if accessory, ok := component["accessory"].(map[string]any); ok {
		normalizeComponentValue(accessory)
	}
	if media, ok := component["media"].(map[string]any); ok {
		normalizeMediaValue(media)
	}
	if items, ok := component["items"].([]any); ok {
		for _, item := range items {
			if object, objectOK := item.(map[string]any); objectOK {
				deleteFalseComponentField(object, "spoiler")
			}
		}
	}
}

func normalizeMediaValue(media map[string]any) {
	url, ok := media["url"].(string)
	if !ok {
		return
	}
	clear(media)
	media["url"] = url
}

func deleteFalseComponentField(component map[string]any, field string) {
	if value, ok := component[field].(bool); ok && !value {
		delete(component, field)
	}
}

func (state State) valid() bool {
	return state == StatePending || state == StateHealthy || state == StateDrifted || state == StateRepairing || state == StateBlocked || state == StateArchived
}

// Hash returns the canonical observable Components V2 SHA-256 digest.
func (payload Payload) Hash() (string, error) {
	encoded, err := json.Marshal(payload.Normalize().Components)
	if err != nil {
		return "", fmt.Errorf("marshal canonical message: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
