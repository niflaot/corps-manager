package agreements

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRenderProducesOneListPerCompanyWithChannel(t *testing.T) {
	t.Parallel()
	companies := []Company{{ID: "rage", Name: "Rage", ChannelID: "111"}, {ID: "lnt", Name: "LNT", ChannelID: "222"},
		{ID: "legacy", Name: "Legacy"}}
	items := []Agreement{{CompanyID: "rage", ID: "lspd", Description: "Descuento para sus integrantes.",
		ImageURL: "https://example.com/lspd.png"}, {CompanyID: "lnt", ID: "otro", Description: "Otro convenio."}}
	definitions, err := Render(companies, items, Config{ControlChannelID: "333"}, "987654321098765432")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(definitions) != 3 {
		t.Fatalf("len(definitions) = %d, want 3", len(definitions))
	}
	channels := map[string]string{}
	for _, definition := range definitions {
		if err := definition.Validate(); err != nil {
			t.Fatalf("definition %q Validate: %v", definition.Key, err)
		}
		channels[definition.Key] = definition.ChannelID
	}
	if channels["agr-rage"] != "111" || channels["agr-lnt"] != "222" || channels[agreementsControlMessageKey] != "333" {
		t.Fatalf("channels = %v", channels)
	}
}

func TestMaximumAgreementDescriptionsFitDiscord(t *testing.T) {
	items := make([]Agreement, 20)
	for index := range items {
		items[index] = Agreement{CompanyID: "company", ID: strings.Repeat("b", 64), Description: strings.Repeat("D", 1000)}
	}
	company := Company{ID: strings.Repeat("c", 60), Name: strings.Repeat("A", 80), ChannelID: "123"}
	for index := range items {
		items[index].CompanyID = company.ID
	}
	definitions, err := Render([]Company{company}, items, Config{ControlChannelID: "456"}, "789")
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
		if _, err := service.CreateCompany(context.Background(), item[0], item[1], "123"); !errors.Is(err, ErrInvalidCompany) {
			t.Fatalf("accepted invalid company: %v", err)
		}
	}
	if _, err := service.CreateCompany(context.Background(), "company", "Valid company", "not-a-channel"); !errors.Is(err, ErrInvalidCompany) {
		t.Fatalf("accepted invalid channel: %v", err)
	}
	if _, err := service.CreateCompany(context.Background(), strings.Repeat("c", 61), "Valid company", "123"); !errors.Is(err, ErrInvalidCompany) {
		t.Fatalf("accepted oversized company id: %v", err)
	}
	if _, err := service.Create(context.Background(), "missing space", "id", "Description", "", "123"); !errors.Is(err, ErrInvalidCompany) {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), "company", "id", "Description", "http://example.com/image.png", "123"); !errors.Is(err, ErrInvalidImageURL) {
		t.Fatal(err)
	}
}

func TestAgreementEditAndDeleteValidateInputs(t *testing.T) {
	service := NewService(Config{Enabled: true}, nil, nil, "123")
	if _, err := service.UpdateAgreement(context.Background(), "Bad Company", "id", "Description", ""); !errors.Is(err, ErrInvalidID) {
		t.Fatal(err)
	}
	if _, err := service.UpdateAgreement(context.Background(), "company", "id", "x", ""); !errors.Is(err, ErrInvalidDescription) {
		t.Fatal(err)
	}
	if _, err := service.UpdateAgreement(context.Background(), "company", "id", "Description", "http://example.com/a.png"); !errors.Is(err, ErrInvalidImageURL) {
		t.Fatal(err)
	}
	if err := service.DeleteAgreement(context.Background(), "company", "Bad Id"); !errors.Is(err, ErrInvalidID) {
		t.Fatal(err)
	}
	if err := NewService(Config{}, nil, nil, "123").DeleteAgreement(context.Background(), "company", "id"); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
}
