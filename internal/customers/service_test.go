package customers

import "testing"

func TestNormalizeName(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"Jane Doe":        "jane_doe",
		"  JOHN__SMITH  ": "john_smith",
		"María-José":      "maría_josé",
	}
	for input, expected := range tests {
		actual, err := NormalizeName(input)
		if err != nil {
			t.Fatalf("NormalizeName(%q): %v", input, err)
		}
		if actual != expected {
			t.Fatalf("NormalizeName(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestNormalizeNameRejectsPunctuation(t *testing.T) {
	t.Parallel()
	if _, err := NormalizeName("Jane@Doe"); err != ErrInvalidName {
		t.Fatalf("NormalizeName error = %v, want %v", err, ErrInvalidName)
	}
}

func TestRenderProducesValidManagedMessage(t *testing.T) {
	t.Parallel()
	definition, err := Render([]Customer{{Name: "jane_doe", Visits: 3}},
		Config{ChannelID: "123456789012345678"}, "987654321098765432")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("definition.Validate: %v", err)
	}
}
