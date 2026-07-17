package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	var output bytes.Buffer
	if err := Execute(context.Background(), []string{"--version"}, &output, "1.0.0"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.String() != "discord-bot v1.0.0\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestExecuteRejectsUnknownCommand(t *testing.T) {
	err := Execute(context.Background(), []string{"unknown"}, &bytes.Buffer{}, "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("Execute() error = %v", err)
	}
}
