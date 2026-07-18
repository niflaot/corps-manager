package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type definition struct {
	defaultValue json.RawMessage
	validate     func(json.RawMessage) error
}

// Service validates and resolves persisted settings with code defaults.
type Service struct {
	repository  Repository
	definitions map[Key]definition
}

// NewService creates the settings service and its supported key registry.
func NewService(repository Repository, trapWarning ...string) *Service {
	defaultWarning := "Anti-bot protection"
	if len(trapWarning) > 0 && strings.TrimSpace(trapWarning[0]) != "" {
		defaultWarning = trapWarning[0]
	}
	stringValue := func(maximum int, allowEmpty bool) func(json.RawMessage) error {
		return func(raw json.RawMessage) error {
			var value string
			if json.Unmarshal(raw, &value) != nil || (!allowEmpty && strings.TrimSpace(value) == "") || len(value) > maximum {
				return fmt.Errorf("value must be a string up to %d characters", maximum)
			}
			return nil
		}
	}
	snowflake := func(raw json.RawMessage) error {
		var value string
		if json.Unmarshal(raw, &value) != nil || value != "" && !snowflakePattern.MatchString(value) {
			return fmt.Errorf("value must be an empty string or Discord snowflake")
		}
		return nil
	}
	return &Service{repository: repository, definitions: map[Key]definition{
		VerificationMessageKey:      {jsonString(DefaultVerificationMessageKey), stringValue(64, false)},
		VerificationTrapChannelID:   {jsonString(""), snowflake},
		VerificationTrapMessageID:   {jsonString(""), snowflake},
		VerificationTrapChannelName: {jsonString(DefaultVerificationTrapChannelName), stringValue(100, false)},
		VerificationTrapWarning:     {jsonString(defaultWarning), stringValue(4000, false)},
	}}
}

// Get returns one effective setting, including its code default when unset.
func (service *Service) Get(ctx context.Context, key Key) (Record, error) {
	definition, ok := service.definitions[key]
	if !ok || !dottedKeyPattern.MatchString(string(key)) {
		return Record{}, fmt.Errorf("%w: unsupported key", ErrInvalid)
	}
	record, err := service.repository.Get(ctx, key)
	if err == ErrNotFound {
		return Record{Key: key, Value: definition.defaultValue, Default: true}, nil
	}
	return record, err
}

// List returns all effective supported settings.
func (service *Service) List(ctx context.Context) ([]Record, error) {
	result := make([]Record, 0, len(service.definitions))
	keys := make([]string, 0, len(service.definitions))
	for key := range service.definitions {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	for _, rawKey := range keys {
		key := Key(rawKey)
		record, err := service.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, nil
}

// Set validates and persists one supported setting.
func (service *Service) Set(ctx context.Context, key Key, value json.RawMessage, revision uint64) (Record, error) {
	definition, ok := service.definitions[key]
	if !ok || !dottedKeyPattern.MatchString(string(key)) {
		return Record{}, fmt.Errorf("%w: unsupported key", ErrInvalid)
	}
	if err := definition.validate(value); err != nil {
		return Record{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return service.repository.Set(ctx, key, value, revision)
}

// Reset removes one override and returns its effective code default.
func (service *Service) Reset(ctx context.Context, key Key, revision uint64) (Record, error) {
	if _, ok := service.definitions[key]; !ok {
		return Record{}, fmt.Errorf("%w: unsupported key", ErrInvalid)
	}
	if err := service.repository.Reset(ctx, key, revision); err != nil && err != ErrNotFound {
		return Record{}, err
	}
	return service.Get(ctx, key)
}

// String returns one effective string setting.
func (service *Service) String(ctx context.Context, key Key) (string, error) {
	record, err := service.Get(ctx, key)
	if err != nil {
		return "", err
	}
	var value string
	err = json.Unmarshal(record.Value, &value)
	return value, err
}

func jsonString(value string) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}
