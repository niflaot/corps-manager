package inactivity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const publishAttempts = 3

var employeeNamePattern = regexp.MustCompile(`^[A-Za-z]+_[A-Za-z]+$`)

// Service manages inactivity entries and their Discord dashboard.
type Service struct {
	config     Config
	repository Repository
	messages   *messages.Service
	guildID    string
}

// NewService creates the inactivity registry application service.
func NewService(config Config, repository Repository, messageService *messages.Service, guildID string) *Service {
	return &Service{config: config, repository: repository, messages: messageService, guildID: guildID}
}

// List returns every registered employee.
func (service *Service) List(ctx context.Context) ([]Entry, error) {
	if !service.config.Enabled {
		return nil, ErrDisabled
	}
	return service.repository.List(ctx)
}

// Add validates and registers one dismissed employee.
func (service *Service) Add(ctx context.Context, name string, actor string) (Entry, error) {
	if !service.config.Enabled {
		return Entry{}, ErrDisabled
	}
	display, normalized, err := normalizeName(name)
	if err != nil {
		return Entry{}, err
	}
	entry, err := service.repository.Add(ctx, normalized, display, strings.TrimSpace(actor))
	if err != nil {
		return Entry{}, err
	}
	return entry, service.Publish(ctx)
}

// Remove deletes one dismissed employee from the registry.
func (service *Service) Remove(ctx context.Context, name string) error {
	if !service.config.Enabled {
		return ErrDisabled
	}
	_, normalized, err := normalizeName(name)
	if err != nil {
		return err
	}
	if err := service.repository.Remove(ctx, normalized); err != nil {
		return err
	}
	return service.Publish(ctx)
}

// Publish creates or updates the interactive registry message.
func (service *Service) Publish(ctx context.Context) error {
	if !service.config.Enabled {
		return ErrDisabled
	}
	entries, err := service.repository.List(ctx)
	if err != nil {
		return fmt.Errorf("list inactivity dismissals: %w", err)
	}
	definition, err := Render(entries, service.config, service.guildID)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(fmt.Sprint(entries)))
	payloadKey := hex.EncodeToString(digest[:8])
	for attempt := 0; attempt < publishAttempts; attempt++ {
		record, getErr := service.messages.Get(ctx, service.config.MessageKey)
		if errors.Is(getErr, messages.ErrNotFound) {
			_, createErr := service.messages.Create(ctx, definition, "inactivity-create-"+payloadKey)
			if errors.Is(createErr, messages.ErrConflict) {
				continue
			}
			return createErr
		}
		if getErr != nil {
			return getErr
		}
		_, replaceErr := service.messages.Replace(ctx, record.Key, record.Revision, definition,
			fmt.Sprintf("inactivity-replace-%d-%s", record.Revision, payloadKey))
		if errors.Is(replaceErr, messages.ErrConflict) {
			continue
		}
		return replaceErr
	}
	return messages.ErrConflict
}

func normalizeName(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if !employeeNamePattern.MatchString(value) {
		return "", "", ErrInvalidName
	}
	return value, strings.ToLower(value), nil
}
