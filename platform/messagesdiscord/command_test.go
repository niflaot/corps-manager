package messagesdiscord

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/corps-manager/internal/messages"
	"go.uber.org/zap"
)

func TestDefinitionFitsDiscordLimits(t *testing.T) {
	definition := (&command{}).definition()
	if definition.Name != commandName || len(definition.Options) != 6 {
		t.Fatalf("unexpected definition: %+v", definition)
	}
	for _, option := range definition.Options {
		if len(option.Name) > 32 || len(option.Description) > 100 {
			t.Fatalf("option %s exceeds Discord limits", option.Name)
		}
	}
}

func TestFetcherRejectsForeignOrigins(t *testing.T) {
	fetch := httpFetcher(nil)
	for _, target := range []string{"http://cdn.discordapp.com/a.json", "https://example.com/a.json", "https://cdn.discordapp.com.evil.test/a.json", "::"} {
		if _, err := fetch(context.Background(), target); err == nil {
			t.Fatalf("%s accepted", target)
		}
	}
}

func TestDescribeMapsDomainErrors(t *testing.T) {
	c := &command{log: zap.NewNop()}
	for _, err := range []error{messages.ErrNotFound, messages.ErrConflict, messages.ErrInvalidAssignment, messages.ErrForbidden, errors.New("boom")} {
		if c.describe(err, "") == "" {
			t.Fatalf("empty description for %v", err)
		}
	}
	if got := c.describe(nil, "fallback"); got != "fallback" {
		t.Fatal(got)
	}
}
