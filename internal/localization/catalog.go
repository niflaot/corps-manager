// Package localization loads the bot's fast immutable message catalog.
package localization

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pixelados-net/discord-bot/locales"
)

const (
	// VerificationSuccessKey identifies the successful verification message.
	VerificationSuccessKey = "verification.success"
	// VerificationSuccessShortKey identifies the interaction success message.
	VerificationSuccessShortKey = "verification.success_short"
	// VerificationUnverifyKey identifies the unverify button label.
	VerificationUnverifyKey = "verification.unverify"
	// VerificationRemovedKey identifies the successful removal message.
	VerificationRemovedKey = "verification.removed"
	// VerificationFailedKey identifies the interaction failure message.
	VerificationFailedKey = "verification.failed"
	// VerificationProcessingKey identifies the immediate interaction acknowledgement.
	VerificationProcessingKey = "verification.processing"
	// VerificationUnavailableKey identifies an empty verification group state.
	VerificationUnavailableKey = "verification.unavailable"
	// VerificationTrapWarningKey identifies the trap warning message.
	VerificationTrapWarningKey = "verification.trap.warning"
)

// Catalog is an immutable localized string lookup.
type Catalog struct{ messages map[string]string }

// Load parses an optional external catalog or the embedded repository default.
func Load(path string) (*Catalog, error) {
	data := locales.Default
	if strings.TrimSpace(path) != "" {
		loaded, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read localization catalog: %w", err)
		}
		if err == nil {
			data = loaded
		}
	}
	messages := map[string]string{}
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("parse localization catalog: %w", err)
	}
	for _, key := range []string{VerificationSuccessKey, VerificationSuccessShortKey, VerificationUnverifyKey, VerificationRemovedKey, VerificationFailedKey, VerificationProcessingKey, VerificationUnavailableKey, VerificationTrapWarningKey} {
		if strings.TrimSpace(messages[key]) == "" {
			return nil, fmt.Errorf("localization key %s is required", key)
		}
	}
	return &Catalog{messages: messages}, nil
}

// Text returns one localized message with named placeholder substitution.
func (catalog *Catalog) Text(key string, values ...string) string {
	message := catalog.messages[key]
	for index := 0; index+1 < len(values); index += 2 {
		message = strings.ReplaceAll(message, "{"+values[index]+"}", values[index+1])
	}
	return message
}
