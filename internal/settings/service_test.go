package settings

import (
	"context"
	"encoding/json"
	"testing"
)

type settingRepository struct{ records map[Key]Record }

func (repository *settingRepository) Get(_ context.Context, key Key) (Record, error) {
	record, ok := repository.records[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	return record, nil
}
func (repository *settingRepository) List(context.Context) ([]Record, error) { return nil, nil }
func (repository *settingRepository) Set(_ context.Context, key Key, value []byte, revision uint64) (Record, error) {
	record := Record{Key: key, Value: append(json.RawMessage(nil), value...), Revision: revision + 1}
	repository.records[key] = record
	return record, nil
}
func (repository *settingRepository) Reset(_ context.Context, key Key, _ uint64) error {
	delete(repository.records, key)
	return nil
}

func TestServiceResolvesSetsAndResetsTypedDefaults(t *testing.T) {
	service := NewService(&settingRepository{records: map[Key]Record{}}, "localized warning")
	record, err := service.Get(context.Background(), VerificationTrapWarning)
	if err != nil || !record.Default || string(record.Value) != `"localized warning"` {
		t.Fatalf("Get() = %#v, error = %v", record, err)
	}
	updated, err := service.Set(context.Background(), VerificationMessageKey, json.RawMessage(`"welcome"`), 0)
	if err != nil || updated.Revision != 1 {
		t.Fatalf("Set() = %#v, error = %v", updated, err)
	}
	reset, err := service.Reset(context.Background(), VerificationMessageKey, 1)
	if err != nil || !reset.Default || string(reset.Value) != `"verification"` {
		t.Fatalf("Reset() = %#v, error = %v", reset, err)
	}
}

func TestServiceRejectsUnknownAndInvalidValues(t *testing.T) {
	service := NewService(&settingRepository{records: map[Key]Record{}})
	for key, value := range map[Key]json.RawMessage{"unknown.key": json.RawMessage(`true`), VerificationTrapChannelID: json.RawMessage(`"not-id"`)} {
		if _, err := service.Set(context.Background(), key, value, 0); err == nil {
			t.Fatalf("Set(%s) error = nil", key)
		}
	}
}
