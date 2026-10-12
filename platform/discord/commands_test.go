package discord

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func commandEvent(kind discordgo.InteractionType, data discordgo.InteractionData) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: kind, Data: data}}
}

func TestRouterRoutesByNameAndPrefixAndRestrictsDefinitions(t *testing.T) {
	router := NewRouter(context.Background(), "123", []Command{
		{Definition: &discordgo.ApplicationCommand{Name: "alpha", Description: "a"}, ComponentPrefix: "alpha:"},
		{Definition: &discordgo.ApplicationCommand{Name: "beta", Description: "b"}},
	}, nil)
	cases := []struct {
		event *discordgo.InteractionCreate
		want  string
	}{
		{commandEvent(discordgo.InteractionApplicationCommand, discordgo.ApplicationCommandInteractionData{Name: "beta"}), "beta"},
		{commandEvent(discordgo.InteractionApplicationCommandAutocomplete, discordgo.ApplicationCommandInteractionData{Name: "alpha"}), "alpha"},
		{commandEvent(discordgo.InteractionMessageComponent, discordgo.MessageComponentInteractionData{CustomID: "alpha:ok"}), "alpha"},
		{commandEvent(discordgo.InteractionModalSubmit, discordgo.ModalSubmitInteractionData{CustomID: "alpha:m"}), "alpha"},
		{commandEvent(discordgo.InteractionMessageComponent, discordgo.MessageComponentInteractionData{CustomID: "other:ok"}), ""},
		{commandEvent(discordgo.InteractionApplicationCommand, discordgo.ApplicationCommandInteractionData{Name: "gamma"}), ""},
	}
	for _, test := range cases {
		command, found := router.route(test.event)
		if found != (test.want != "") || found && command.Definition.Name != test.want {
			t.Fatalf("routed to %q (found=%t), want %q", command.Definition.Name, found, test.want)
		}
	}
	for _, definition := range router.Definitions() {
		if definition.DefaultMemberPermissions == nil || *definition.DefaultMemberPermissions != discordgo.PermissionAdministrator {
			t.Fatalf("%s is not administrator-only", definition.Name)
		}
	}
}
