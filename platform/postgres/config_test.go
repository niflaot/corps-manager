package postgres

import (
	"strings"
	"testing"
)

func TestDSNAndMaskedDSN(t *testing.T) {
	config := Config{Host: "localhost", Port: 5432, Database: "bot", User: "user", Password: "secret", SSLMode: "disable"}
	if !strings.Contains(config.DSN(), "secret") {
		t.Fatalf("DSN() = %q", config.DSN())
	}
	if strings.Contains(config.MaskedDSN(), "secret") {
		t.Fatalf("MaskedDSN() = %q", config.MaskedDSN())
	}
}
