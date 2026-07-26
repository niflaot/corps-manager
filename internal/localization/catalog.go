// Package localization loads the bot's fast immutable message catalog.
package localization

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pixelados-net/discord-bot/locales"
)

const (
	// VerificationGroupKeyPrefix namespaces localized verification group names.
	VerificationGroupKeyPrefix = "verification.group."
	// VerificationSuccessKey identifies the successful verification message.
	VerificationSuccessKey = "verification.success"
	// VerificationSuccessShortKey identifies the interaction success message.
	VerificationSuccessShortKey = "verification.success_short"
	// VerificationUnverifyKey identifies the unverify button label.
	VerificationUnverifyKey = "verification.unverify"
	// VerificationRemovedKey identifies the successful removal message.
	VerificationRemovedKey = "verification.removed"
	// VerificationUnverifiedKey identifies the unverified direct message.
	VerificationUnverifiedKey = "verification.unverified"
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
type Catalog struct {
	// messages contains the immutable keyed text snapshot.
	messages map[string]string
}

// Load parses an optional external catalog or the embedded repository default.
func Load(ctx context.Context, config Config) (*Catalog, error) {
	data, err := loadCatalogData(ctx, config)
	if err != nil {
		return nil, err
	}
	messages := map[string]string{}
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("parse localization catalog: %w", err)
	}
	for _, key := range []string{VerificationSuccessKey, VerificationSuccessShortKey, VerificationUnverifyKey, VerificationRemovedKey, VerificationUnverifiedKey, VerificationFailedKey, VerificationProcessingKey, VerificationUnavailableKey, VerificationTrapWarningKey} {
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

// GroupName resolves a localized group name by its stable group key.
func (catalog *Catalog) GroupName(key string) string {
	name := strings.TrimSpace(catalog.messages[VerificationGroupKeyPrefix+key])
	if name == "" {
		return key
	}
	return name
}

// loadCatalogData selects and reads the configured catalog source.
func loadCatalogData(ctx context.Context, config Config) ([]byte, error) {
	path, sourceURL := strings.TrimSpace(config.Path), strings.TrimSpace(config.URL)
	if path != "" && sourceURL != "" {
		return nil, fmt.Errorf("only one localization path or URL may be configured")
	}
	if config.HTTPTimeout <= 0 || config.MaxBytes <= 0 {
		return nil, fmt.Errorf("localization timeout and maximum bytes must be positive")
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read localization catalog: %w", err)
		}
		return boundedCatalog(data, config.MaxBytes)
	}
	if sourceURL != "" {
		return loadRemoteCatalog(ctx, sourceURL, config)
	}
	return boundedCatalog(locales.Default, config.MaxBytes)
}

// loadRemoteCatalog reads one bounded HTTP or HTTPS catalog.
func loadRemoteCatalog(ctx context.Context, source string, config Config) ([]byte, error) {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("localization URL must use HTTP or HTTPS")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create localization request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := (&http.Client{Timeout: config.HTTPTimeout}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch localization catalog: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch localization catalog: unexpected HTTP status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, config.MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read remote localization catalog: %w", err)
	}
	return boundedCatalog(data, config.MaxBytes)
}

// boundedCatalog rejects catalog content above the configured maximum.
func boundedCatalog(data []byte, maximum int64) ([]byte, error) {
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("localization catalog exceeds %d bytes", maximum)
	}
	return data, nil
}
