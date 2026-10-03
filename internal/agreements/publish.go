package agreements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/niflaot/corps-manager/internal/messages"
)

const publishAttempts = 3
const createKeyPrefix = "agreement-create-"
const replaceKeyPrefix = "agreement-replace"
const archiveKeyPrefix = "agreement-archive"

// Publish creates or updates every company list and the control panel, archiving obsolete messages.
func (service *Service) Publish(ctx context.Context) error {
	if !service.config.Enabled {
		return ErrDisabled
	}
	companies, err := service.repository.ListCompanies(ctx)
	if err != nil {
		return fmt.Errorf("list companies for panel: %w", err)
	}
	items, err := service.repository.List(ctx)
	if err != nil {
		return fmt.Errorf("list agreements for panel: %w", err)
	}
	definitions, err := Render(companies, items, service.config, service.guildID)
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		if err := service.publishDefinition(ctx, definition); err != nil {
			return err
		}
	}
	return service.archive(ctx, legacyListMessageKey)
}

func (service *Service) archive(ctx context.Context, key string) error {
	for attempt := 0; attempt < publishAttempts; attempt++ {
		record, err := service.messages.Get(ctx, key)
		if errors.Is(err, messages.ErrNotFound) || (err == nil && record.State == messages.StateArchived) {
			return nil
		}
		if err == nil {
			idempotency := fmt.Sprintf("%s-%s-%d", archiveKeyPrefix, key, record.Revision)
			_, err = service.messages.Archive(ctx, key, record.Revision, idempotency)
		}
		if !errors.Is(err, messages.ErrConflict) {
			return err
		}
	}
	return messages.ErrConflict
}

func (service *Service) publishDefinition(ctx context.Context, definition messages.Definition) error {
	encoded, err := json.Marshal(definition)
	if err != nil {
		return fmt.Errorf("encode agreement panel fingerprint: %w", err)
	}
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:8])
	for attempt := 0; attempt < publishAttempts; attempt++ {
		record, getErr := service.messages.Get(ctx, definition.Key)
		if errors.Is(getErr, messages.ErrNotFound) {
			_, err = service.messages.Create(ctx, definition, createKeyPrefix+definition.Key+"-"+fingerprint)
		} else if getErr == nil {
			key := fmt.Sprintf("%s-%s-%d-%s", replaceKeyPrefix, definition.Key, record.Revision, fingerprint)
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
