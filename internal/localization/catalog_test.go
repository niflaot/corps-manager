package localization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadEmbeddedCatalogResolvesGroupNamesByKey verifies embedded translation and key fallback.
func TestLoadEmbeddedCatalogResolvesGroupNamesByKey(t *testing.T) {
	catalog, err := Load(context.Background(), testConfig())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if name := catalog.GroupName("member"); name != "Miembro" {
		t.Fatalf("GroupName(member) = %q", name)
	}
	if name := catalog.GroupName("custom"); name != "custom" {
		t.Fatalf("GroupName(custom) = %q", name)
	}
	if text := catalog.Text(VerificationSuccessKey, "group", "Miembro"); text != "✅ Te verificaste correctamente como **Miembro**." {
		t.Fatalf("Text() = %q", text)
	}
}

// TestLoadCatalogFromFile verifies a configured local source.
func TestLoadCatalogFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, testCatalogJSON("Archivo"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	config := testConfig()
	config.Path = path
	catalog, err := Load(context.Background(), config)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if name := catalog.GroupName("member"); name != "Archivo" {
		t.Fatalf("GroupName(member) = %q", name)
	}
}

// TestLoadCatalogFromURL verifies a bounded remote source.
func TestLoadCatalogFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", request.Header.Get("Accept"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(testCatalogJSON("Remoto"))
	}))
	defer server.Close()
	config := testConfig()
	config.URL = server.URL
	catalog, err := Load(context.Background(), config)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if name := catalog.GroupName("member"); name != "Remoto" {
		t.Fatalf("GroupName(member) = %q", name)
	}
}

// TestLoadRejectsAmbiguousAndOversizedSources verifies source safety constraints.
func TestLoadRejectsAmbiguousAndOversizedSources(t *testing.T) {
	config := testConfig()
	config.Path, config.URL = "messages.json", "https://example.com/messages.json"
	if _, err := Load(context.Background(), config); err == nil {
		t.Fatal("Load() ambiguous source error = nil")
	}
	config = testConfig()
	config.MaxBytes = 1
	if _, err := Load(context.Background(), config); err == nil {
		t.Fatal("Load() oversized source error = nil")
	}
}

// TestLoadRejectsInvalidExternalCatalogs verifies startup failures for unusable overrides.
func TestLoadRejectsInvalidExternalCatalogs(t *testing.T) {
	tests := []struct {
		name   string
		config func(*testing.T) Config
	}{
		{name: "missing file", config: func(t *testing.T) Config {
			config := testConfig()
			config.Path = filepath.Join(t.TempDir(), "missing.json")
			return config
		}},
		{name: "invalid URL", config: func(*testing.T) Config {
			config := testConfig()
			config.URL = "file:///messages.json"
			return config
		}},
		{name: "HTTP failure", config: func(t *testing.T) Config {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				http.Error(writer, "unavailable", http.StatusServiceUnavailable)
			}))
			t.Cleanup(server.Close)
			config := testConfig()
			config.URL = server.URL
			return config
		}},
		{name: "invalid JSON", config: func(t *testing.T) Config {
			path := filepath.Join(t.TempDir(), "messages.json")
			if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			config := testConfig()
			config.Path = path
			return config
		}},
		{name: "missing required key", config: func(t *testing.T) Config {
			path := filepath.Join(t.TempDir(), "messages.json")
			if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			config := testConfig()
			config.Path = path
			return config
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Load(context.Background(), test.config(t)); err == nil {
				t.Fatal("Load() error = nil")
			}
		})
	}
}

// TestLoadConfigProvidesSafeDefaults verifies localization environment defaults.
func TestLoadConfigProvidesSafeDefaults(t *testing.T) {
	t.Setenv("DISCORD_BOT_LOCALES_PATH", "")
	t.Setenv("DISCORD_BOT_LOCALES_URL", "")
	t.Setenv("DISCORD_BOT_LOCALES_HTTP_TIMEOUT", "")
	t.Setenv("DISCORD_BOT_LOCALES_MAX_BYTES", "")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.Path != "" || config.URL != "" || config.HTTPTimeout != 10*time.Second || config.MaxBytes != 1<<20 {
		t.Fatalf("config = %#v", config)
	}
}

// testConfig creates bounded deterministic catalog settings.
func testConfig() Config {
	return Config{HTTPTimeout: time.Second, MaxBytes: 1 << 20}
}

// testCatalogJSON creates a complete external test catalog.
func testCatalogJSON(group string) []byte {
	return []byte(`{
		"verification.group.member":"` + group + `",
		"verification.success":"verified {group}",
		"verification.success_short":"verified",
		"verification.unverify":"unverify",
		"verification.removed":"removed",
		"verification.unverified":"unverified {group}",
		"verification.failed":"failed",
		"verification.processing":"processing",
		"verification.unavailable":"unavailable",
		"verification.trap.warning":"warning"
	}`)
}
