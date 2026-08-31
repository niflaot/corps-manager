package agreements

import "testing"

func TestRenderProducesValidManagedMessages(t *testing.T) {
	t.Parallel()
	definitions, err := Render([]Agreement{{ID: "lspd", Description: "Descuento para sus integrantes.",
		ImageURL: "https://example.com/lspd.png"}}, Config{ChannelID: "123456789012345678",
		ControlChannelID: "234567890123456789"}, "987654321098765432")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(definitions) != 2 {
		t.Fatalf("len(definitions) = %d, want 2", len(definitions))
	}
	for _, definition := range definitions {
		if err := definition.Validate(); err != nil {
			t.Fatalf("definition %q Validate: %v", definition.Key, err)
		}
	}
}
