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

// EmbedMedia contains an embed media URL.
type EmbedMedia struct {
	// URL is the public media URL.
	URL string `json:"url"`
}

// EmbedAuthor contains embed author data.
type EmbedAuthor struct {
	// Name is the author label.
	Name string `json:"name"`
	// URL is the optional author link.
	URL string `json:"url,omitempty"`
	// IconURL is the optional author icon.
	IconURL string `json:"iconUrl,omitempty"`
}

// EmbedFooter contains embed footer data.
type EmbedFooter struct {
	// Text is the footer label.
	Text string `json:"text"`
	// IconURL is the optional footer icon.
	IconURL string `json:"iconUrl,omitempty"`
}

// EmbedField contains one ordered embed field.
type EmbedField struct {
	// Name is the field label.
	Name string `json:"name"`
	// Value is the field body.
	Value string `json:"value"`
	// Inline requests inline rendering.
	Inline bool `json:"inline,omitempty"`
}

// Embed contains one managed rich embed.
type Embed struct {
	// Title is the embed title.
	Title string `json:"title,omitempty"`
	// Description is the embed body.
	Description string `json:"description,omitempty"`
	// URL is the optional title link.
	URL string `json:"url,omitempty"`
	// Timestamp is an optional RFC3339 timestamp.
	Timestamp string `json:"timestamp,omitempty"`
	// Color is an optional RGB integer.
	Color int `json:"color,omitempty"`
	// Footer contains optional footer data.
	Footer *EmbedFooter `json:"footer,omitempty"`
	// Image contains an optional large image.
	Image *EmbedMedia `json:"image,omitempty"`
	// Thumbnail contains an optional thumbnail.
	Thumbnail *EmbedMedia `json:"thumbnail,omitempty"`
	// Author contains optional author data.
	Author *EmbedAuthor `json:"author,omitempty"`
	// Fields contains ordered embed fields.
	Fields []EmbedField `json:"fields"`
}

// Payload contains the managed Discord message body.
type Payload struct {
	// Content is optional message text.
	Content string `json:"content"`
	// Embeds contains up to ten ordered embeds.
	Embeds []Embed `json:"embeds"`
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
	payload.Embeds = slices.Clone(payload.Embeds)
	if payload.Embeds == nil {
		payload.Embeds = []Embed{}
	}
	for index := range payload.Embeds {
		if parsed, err := time.Parse(time.RFC3339, payload.Embeds[index].Timestamp); err == nil {
			payload.Embeds[index].Timestamp = parsed.Format(time.RFC3339Nano)
		}
		payload.Embeds[index].Fields = slices.Clone(payload.Embeds[index].Fields)
		if payload.Embeds[index].Fields == nil {
			payload.Embeds[index].Fields = []EmbedField{}
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
	if len(payload.AllowedMentions.Roles) == 0 {
		payload.AllowedMentions.Roles = nil
	}
	if len(payload.AllowedMentions.Users) == 0 {
		payload.AllowedMentions.Users = nil
	}
	return payload
}

func (state State) valid() bool {
	switch state {
	case StatePending, StateHealthy, StateDrifted, StateRepairing, StateBlocked, StateArchived:
		return true
	default:
		return false
	}
}

// Hash returns the canonical observable payload SHA-256 digest.
func (payload Payload) Hash() (string, error) {
	payload = payload.Normalize()
	observable := struct {
		Content string  `json:"content"`
		Embeds  []Embed `json:"embeds"`
	}{Content: payload.Content, Embeds: payload.Embeds}
	encoded, err := json.Marshal(observable)
	if err != nil {
		return "", fmt.Errorf("marshal canonical message: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
