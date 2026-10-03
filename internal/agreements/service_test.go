package agreements

import (
	"context"
	"errors"
	"strings"
	"testing"
)

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

func TestMaximumAgreementDescriptionsFitDiscord(t *testing.T) {
	items := make([]Agreement, 20)
	for index := range items {
		items[index] = Agreement{CompanyID: "company", CompanyName: strings.Repeat("A", 80), ID: strings.Repeat("b", 64), Description: strings.Repeat("D", 1000)}
	}
	definitions, err := Render(items, Config{ChannelID: "123", ControlChannelID: "456"}, "789")
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range definitions {
		if err := definition.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidCompanyAndAgreementInputs(t *testing.T) {
	service := NewService(Config{Enabled: true}, nil, nil, "123")
	for _, item := range [][2]string{{"", "Valid company"}, {"UPPERCASE", "Valid company"}, {"company", ""}, {"company", strings.Repeat("x", 81)}} {
		if _, err := service.CreateCompany(context.Background(), item[0], item[1]); !errors.Is(err, ErrInvalidCompany) {
			t.Fatalf("accepted invalid company: %v", err)
		}
	}
	if _, err := service.Create(context.Background(), "missing space", "id", "Description", "", "123"); !errors.Is(err, ErrInvalidCompany) {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), "company", "id", "Description", "http://example.com/image.png", "123"); !errors.Is(err, ErrInvalidImageURL) {
		t.Fatal(err)
	}
}
