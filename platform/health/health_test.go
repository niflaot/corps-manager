package health

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestSnapshot(t *testing.T) {
	service := New(map[string]Check{
		"available":   func(context.Context) error { return nil },
		"unavailable": func(context.Context) error { return errors.New("offline") },
	})
	statuses := service.Snapshot(context.Background())
	if statuses["available"] != StatusAvailable || statuses["unavailable"] != StatusUnavailable {
		t.Fatalf("Snapshot() = %#v", statuses)
	}
}

func TestHealthJobLogsHealthyAtDebugAndFailuresAtError(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	log := zap.New(core)
	healthy := provideHealthJob(New(map[string]Check{"postgres": func(context.Context) error { return nil }}), log)
	if err := healthy.Handler(context.Background()); err != nil {
		t.Fatalf("healthy Handler() error = %v", err)
	}
	unhealthy := provideHealthJob(New(map[string]Check{"postgres": func(context.Context) error {
		return errors.New("offline")
	}}), log)
	if err := unhealthy.Handler(context.Background()); err != nil {
		t.Fatalf("unhealthy Handler() error = %v", err)
	}
	entries := observed.All()
	if len(entries) != 2 || entries[0].Level != zap.DebugLevel || entries[1].Level != zap.ErrorLevel {
		t.Fatalf("health log entries = %#v", entries)
	}
}
