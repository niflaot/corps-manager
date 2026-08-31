package customers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/niflaot/corps-manager/internal/messages"
)

const publishAttempts = 3

// Service manages frequent customers and their Discord panel.
type Service struct {
	config     Config
	repository Repository
	messages   *messages.Service
	guildID    string
}

// NewService creates the customer application service.
func NewService(config Config, repository Repository, messageService *messages.Service, guildID string) *Service {
	return &Service{config: config, repository: repository, messages: messageService, guildID: guildID}
}

// NormalizeName converts a customer name to lowercase underscore form.
func NormalizeName(value string) (string, error) {
	var result strings.Builder
	pendingSeparator := false
	for _, current := range strings.TrimSpace(value) {
		switch {
		case unicode.IsLetter(current) || unicode.IsDigit(current):
			if pendingSeparator && result.Len() > 0 {
				result.WriteByte('_')
			}
			pendingSeparator = false
			result.WriteRune(unicode.ToLower(current))
		case unicode.IsSpace(current) || current == '_' || current == '-':
			pendingSeparator = result.Len() > 0
		default:
			return "", ErrInvalidName
		}
	}
	normalized := result.String()
	if len([]rune(normalized)) < 2 || len(normalized) > 64 {
		return "", ErrInvalidName
	}
	return normalized, nil
}

// Record registers one visit made by a Discord attendant.
func (service *Service) Record(ctx context.Context, name string, userID string, displayName string) (Customer, error) {
	if !service.config.Enabled {
		return Customer{}, ErrDisabled
	}
	normalized, err := NormalizeName(name)
	if err != nil {
		return Customer{}, err
	}
	userID, displayName = strings.TrimSpace(userID), strings.TrimSpace(displayName)
	if !snowflakePattern.MatchString(userID) || displayName == "" || len([]rune(displayName)) > 80 {
		return Customer{}, ErrInvalidAttendant
	}
	customer, err := service.repository.Record(ctx, normalized, userID, displayName)
	if err != nil {
		return Customer{}, err
	}
	return customer, service.Publish(ctx)
}

// List returns every frequent customer.
func (service *Service) List(ctx context.Context) ([]Customer, error) {
	if !service.config.Enabled {
		return nil, ErrDisabled
	}
	return service.repository.List(ctx)
}

// Get returns one normalized customer and attendant history.
func (service *Service) Get(ctx context.Context, name string) (Customer, error) {
	if !service.config.Enabled {
		return Customer{}, ErrDisabled
	}
	normalized, err := NormalizeName(name)
	if err != nil {
		return Customer{}, err
	}
	return service.repository.Get(ctx, normalized)
}

// Delete removes one normalized customer and republishes the panel.
func (service *Service) Delete(ctx context.Context, name string) error {
	if !service.config.Enabled {
		return ErrDisabled
	}
	normalized, err := NormalizeName(name)
	if err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, normalized); err != nil {
		return err
	}
	return service.Publish(ctx)
}

// Publish creates or updates the frequent-customer panel.
func (service *Service) Publish(ctx context.Context) error {
	if !service.config.Enabled {
		return ErrDisabled
	}
	customers, err := service.repository.List(ctx)
	if err != nil {
		return fmt.Errorf("list customers for panel: %w", err)
	}
	definition, err := Render(customers, service.config, service.guildID)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(definition)
	if err != nil {
		return fmt.Errorf("encode customer panel fingerprint: %w", err)
	}
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:8])
	for attempt := 0; attempt < publishAttempts; attempt++ {
		record, getErr := service.messages.Get(ctx, definition.Key)
		if errors.Is(getErr, messages.ErrNotFound) {
			_, err = service.messages.Create(ctx, definition, "customer-panel-create-"+fingerprint)
		} else if getErr == nil {
			key := fmt.Sprintf("customer-panel-replace-%d-%s", record.Revision, fingerprint)
			_, err = service.messages.Replace(ctx, record.Key, record.Revision, definition, key)
		} else {
			return getErr
		}
		if !errors.Is(err, messages.ErrConflict) {
			return err
		}
	}
	return messages.ErrConflict
}
