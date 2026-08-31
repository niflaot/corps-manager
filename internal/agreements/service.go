package agreements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const publishAttempts = 3

var agreementIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Service manages agreements and their managed Discord messages.
type Service struct {
	config     Config
	repository Repository
	messages   *messages.Service
	guildID    string
}

// NewService creates the agreements application service.
func NewService(config Config, repository Repository, messageService *messages.Service, guildID string) *Service {
	return &Service{config: config, repository: repository, messages: messageService, guildID: guildID}
}

// Create validates and persists one agreement.
func (service *Service) Create(ctx context.Context, id string, description string, imageURL string,
	actor string) (Agreement, error) {
	if !service.config.Enabled {
		return Agreement{}, ErrDisabled
	}
	id = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), " ", "_"))
	description, imageURL, actor = strings.TrimSpace(description), strings.TrimSpace(imageURL), strings.TrimSpace(actor)
	if !agreementIDPattern.MatchString(id) {
		return Agreement{}, ErrInvalidID
	}
	if len([]rune(description)) < 3 || len([]rune(description)) > 1000 {
		return Agreement{}, ErrInvalidDescription
	}
	if imageURL != "" {
		parsed, err := url.ParseRequestURI(imageURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return Agreement{}, ErrInvalidImageURL
		}
	}
	if !snowflakePattern.MatchString(actor) {
		return Agreement{}, ErrInvalidID
	}
	agreement, err := service.repository.Create(ctx, Agreement{ID: id, Description: description,
		ImageURL: imageURL, CreatedBy: actor})
	if err != nil {
		return Agreement{}, err
	}
	return agreement, service.Publish(ctx)
}

// List returns every configured agreement.
func (service *Service) List(ctx context.Context) ([]Agreement, error) {
	if !service.config.Enabled {
		return nil, ErrDisabled
	}
	return service.repository.List(ctx)
}

// Publish creates or updates the agreement list and its control panel.
func (service *Service) Publish(ctx context.Context) error {
	if !service.config.Enabled {
		return ErrDisabled
	}
	items, err := service.repository.List(ctx)
	if err != nil {
		return fmt.Errorf("list agreements for panel: %w", err)
	}
	definitions, err := Render(items, service.config, service.guildID)
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		if err := service.publishDefinition(ctx, definition); err != nil {
			return err
		}
	}
	return nil
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
			_, err = service.messages.Create(ctx, definition, "agreement-create-"+definition.Key+"-"+fingerprint)
		} else if getErr == nil {
			key := fmt.Sprintf("agreement-replace-%s-%d-%s", definition.Key, record.Revision, fingerprint)
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
