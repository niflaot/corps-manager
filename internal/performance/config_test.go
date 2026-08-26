package performance

import (
	"testing"
	"time"
)

func TestLoadConfigEnabled(t *testing.T) {
	t.Setenv("DISCORD_BOT_PERFORMANCE_ENABLED", "true")
	t.Setenv("DISCORD_BOT_PERFORMANCE_ENDPOINT", "https://example.test/api/query")
	t.Setenv("DISCORD_BOT_PERFORMANCE_BUSINESS_ID", "1995")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CHANNEL_ID", "123456789")
	t.Setenv("DISCORD_BOT_PERFORMANCE_CUTOFF_WEEKDAY", "Tuesday")
	t.Setenv("DISCORD_BOT_PERFORMANCE_TIMEZONE", "America/Bogota")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !config.Enabled || config.BusinessID != 1995 || config.CutoffWeekday != time.Tuesday ||
		config.Interval != 6*time.Hour || config.Timezone.String() != "America/Bogota" {
		t.Fatalf("LoadConfig() = %#v", config)
	}
}

func TestLoadConfigDisabledDoesNotRequireEndpoint(t *testing.T) {
	t.Setenv("DISCORD_BOT_PERFORMANCE_ENABLED", "false")
	t.Setenv("DISCORD_BOT_PERFORMANCE_ENDPOINT", "")
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
}
