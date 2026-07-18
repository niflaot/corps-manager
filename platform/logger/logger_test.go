package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestJSONLoggerHonorsLevel(t *testing.T) {
	var output bytes.Buffer
	log := newWithSink(Config{Level: zapcore.InfoLevel, Format: FormatJSON}, zapcore.AddSync(&output))
	log.Debug("hidden")
	log.Info("visible", zap.String("key", "value"))
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if entry["msg"] != "visible" || entry["key"] != "value" {
		t.Fatalf("entry = %#v", entry)
	}
}
