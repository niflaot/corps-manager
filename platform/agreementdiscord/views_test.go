package agreementdiscord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/agreements"
	"go.uber.org/zap"
)

type companyRepository struct {
	// Repository supplies unused persistence operations.
	agreements.Repository
	companies []agreements.Company
}

func (repository *companyRepository) ListCompanies(context.Context) ([]agreements.Company, error) {
	return repository.companies, nil
}

type transportFunc func(*http.Request) (*http.Response, error)

func (transport transportFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestCompanySelectorPaginatesBeyondDiscordLimit(t *testing.T) {
	repository := &companyRepository{}
	for i := range 30 {
		repository.companies = append(repository.companies, agreements.Company{ID: fmt.Sprintf("company-%02d", i), Name: fmt.Sprintf("Empresa %d", i)})
	}
	handler := &handler{ctx: context.Background(), guildID: "123", service: agreements.NewService(agreements.Config{}, repository, nil, "123"), log: zap.NewNop()}
	session, _ := discordgo.New("Bot test")
	var payload map[string]json.RawMessage
	session.Client = &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"987"}`)), Request: request}, nil
	})}
	interaction := &discordgo.Interaction{ID: "111", AppID: "222", Token: "token"}
	for page, want := range []int{25, 5} {
		handler.showCompanies(context.Background(), session, interaction, agreementMode, page)
		var rows []struct {
			Components []struct {
				Options []discordgo.SelectMenuOption `json:"options"`
			} `json:"components"`
		}
		if err := json.Unmarshal(payload["components"], &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || len(rows[0].Components[0].Options) != want {
			t.Fatalf("page=%d rows=%s", page, payload["components"])
		}
	}
	handler.openModal(context.Background(), session, interaction, "company-29")
	var data struct {
		CustomID   string            `json:"custom_id"`
		Components []json.RawMessage `json:"components"`
	}
	if err := json.Unmarshal(payload["data"], &data); err != nil {
		t.Fatal(err)
	}
	if data.CustomID != addModalPrefix+"company-29" || len(data.Components) != 3 {
		t.Fatalf("modal lost company binding: %+v", data)
	}
}

func TestAgreementHandlerRejectsOtherGuildsAndChannels(t *testing.T) {
	handler := &handler{ctx: context.Background(), guildID: "123", config: agreements.Config{Enabled: true, ControlChannelID: "456"}, log: zap.NewNop()}
	for _, scope := range [][2]string{{"999", "456"}, {"123", "999"}} {
		event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{GuildID: scope[0], ChannelID: scope[1], Type: discordgo.InteractionMessageComponent, Data: discordgo.MessageComponentInteractionData{CustomID: agreements.ButtonAddCustomID}, Member: &discordgo.Member{User: &discordgo.User{ID: "111"}}}}
		// A rejected interaction must not touch either missing dependency.
		handler.handle(nil, event)
	}
}
