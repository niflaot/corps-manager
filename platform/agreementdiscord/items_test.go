package agreementdiscord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/agreements"
	"go.uber.org/zap"
)

type itemRepository struct {
	// Repository supplies unused persistence operations.
	agreements.Repository
	items []agreements.Agreement
}

func (repository *itemRepository) List(context.Context) ([]agreements.Agreement, error) {
	return repository.items, nil
}

func TestItemSelectorListsOnlyTheChosenCompany(t *testing.T) {
	repository := &itemRepository{items: []agreements.Agreement{
		{CompanyID: "rage", CompanyName: "Rage", ID: "lspd", Description: "Descuento"},
		{CompanyID: "lnt", CompanyName: "LNT", ID: "otro", Description: "Otro convenio"},
		{CompanyID: "rage", CompanyName: "Rage", ID: "ems", Description: "Otro descuento"},
	}}
	handler := &handler{ctx: context.Background(), guildID: "123", log: zap.NewNop(),
		service: agreements.NewService(agreements.Config{Enabled: true}, repository, nil, "123")}
	session, _ := discordgo.New("Bot test")
	var payload map[string]json.RawMessage
	session.Client = &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"987"}`)), Request: request}, nil
	})}
	interaction := &discordgo.Interaction{ID: "111", AppID: "222", Token: "token"}
	handler.showItems(context.Background(), session, interaction, itemEditMode, "rage", 0)
	var rows []struct {
		CustomID   string `json:"custom_id"`
		Components []struct {
			CustomID string                       `json:"custom_id"`
			Options  []discordgo.SelectMenuOption `json:"options"`
		} `json:"components"`
	}
	if err := json.Unmarshal(payload["components"], &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].Components[0].Options) != 2 || rows[0].Components[0].CustomID != itemEditSelectPrefix+"rage" {
		t.Fatalf("components = %s", payload["components"])
	}
}
