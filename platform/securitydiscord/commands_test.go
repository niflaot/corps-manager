package securitydiscord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/niflaot/corps-manager/internal/security"
)

func TestVerificationUpdateOnlyCarriesProvidedOptions(t *testing.T) {
	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: optionChannel, Type: discordgo.ApplicationCommandOptionChannel, Value: "456"},
		{Name: optionEnabled, Type: discordgo.ApplicationCommandOptionBoolean, Value: false},
	}
	update := (&commands{}).verificationUpdate(options)
	if update.ChannelID == nil || *update.ChannelID != "456" || update.RoleID != nil || update.MessageKey != nil {
		t.Fatalf("update=%+v", update)
	}
	if update.VerificationEnabled == nil || *update.VerificationEnabled {
		t.Fatal("explicit false was lost")
	}
}

func TestCommandDefinitionsAreNamed(t *testing.T) {
	if newVerificationCommand(&security.Service{}, nil).Definition.Name != verificationCommand || newAntibotCommand(&security.Service{}, nil).Definition.Name != antibotCommand {
		t.Fatal("unexpected command names")
	}
}
