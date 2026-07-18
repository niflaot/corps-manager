package messages

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

const maxComponentCount = 40

var topLevelTypes = []discordgo.ComponentType{1, 9, 10, 12, 13, 14, 17}

// Validate verifies the definition and Discord payload constraints.
func (definition Definition) Validate() error {
	if !keyPattern.MatchString(definition.Key) {
		return fmt.Errorf("invalid message key")
	}
	if !snowflakePattern.MatchString(definition.GuildID) || !snowflakePattern.MatchString(definition.ChannelID) {
		return fmt.Errorf("guildId and channelId must be Discord snowflake strings")
	}
	return definition.Payload.Validate()
}

// Validate verifies Components V2 limits and mention policy.
func (payload Payload) Validate() error {
	payload = payload.Normalize()
	if len(payload.Components) == 0 {
		return fmt.Errorf("at least one component is required")
	}
	total := 0
	for index, raw := range payload.Components {
		if err := validateComponentJSON(raw); err != nil {
			return fmt.Errorf("component %d: %w", index, err)
		}
		component, err := discordgo.MessageComponentFromJSON(raw)
		if err != nil {
			return fmt.Errorf("component %d: %w", index, err)
		}
		if !slices.Contains(topLevelTypes, component.Type()) {
			return fmt.Errorf("component %d: type %d is not valid at top level", index, component.Type())
		}
		count, err := validateComponent(component, true)
		if err != nil {
			return fmt.Errorf("component %d: %w", index, err)
		}
		total += count
	}
	if total > maxComponentCount {
		return fmt.Errorf("components exceed %d total items", maxComponentCount)
	}
	return validateMentions(payload.AllowedMentions)
}

func validateComponent(component discordgo.MessageComponent, topLevel bool) (int, error) {
	switch value := component.(type) {
	case *discordgo.TextDisplay:
		if !topLevel || strings.TrimSpace(value.Content) == "" || utf8.RuneCountInString(value.Content) > 4000 {
			return 0, fmt.Errorf("text display content must contain 1 to 4000 characters")
		}
	case *discordgo.ActionsRow:
		if !topLevel || len(value.Components) == 0 || len(value.Components) > 5 {
			return 0, fmt.Errorf("action row must contain 1 to 5 buttons")
		}
		for _, child := range value.Components {
			button, ok := child.(*discordgo.Button)
			if !ok {
				return 0, fmt.Errorf("action rows currently support buttons only")
			}
			if err := validateButton(button); err != nil {
				return 0, err
			}
		}
		return 1 + len(value.Components), nil
	case *discordgo.Section:
		if !topLevel || len(value.Components) == 0 || len(value.Components) > 3 || value.Accessory == nil {
			return 0, fmt.Errorf("section requires 1 to 3 text displays and one accessory")
		}
		for _, child := range value.Components {
			if _, ok := child.(*discordgo.TextDisplay); !ok {
				return 0, fmt.Errorf("section children must be text displays")
			}
			if _, err := validateComponent(child, true); err != nil {
				return 0, err
			}
		}
		switch accessory := value.Accessory.(type) {
		case *discordgo.Button:
			if err := validateButton(accessory); err != nil {
				return 0, err
			}
		case *discordgo.Thumbnail:
			if strings.TrimSpace(accessory.Media.URL) == "" {
				return 0, fmt.Errorf("thumbnail media URL is required")
			}
		default:
			return 0, fmt.Errorf("section accessory must be a button or thumbnail")
		}
		return 2 + len(value.Components), nil
	case *discordgo.Container:
		if !topLevel || len(value.Components) == 0 || value.AccentColor != nil && (*value.AccentColor < 0 || *value.AccentColor > 0xffffff) {
			return 0, fmt.Errorf("container requires children and a valid accent color")
		}
		count := 1
		for _, child := range value.Components {
			if _, nested := child.(*discordgo.Container); nested {
				return 0, fmt.Errorf("containers cannot be nested")
			}
			childCount, err := validateComponent(child, true)
			if err != nil {
				return 0, err
			}
			count += childCount
		}
		return count, nil
	case *discordgo.MediaGallery:
		if !topLevel || len(value.Items) == 0 || len(value.Items) > 10 {
			return 0, fmt.Errorf("media gallery must contain 1 to 10 items")
		}
		for _, item := range value.Items {
			if strings.TrimSpace(item.Media.URL) == "" {
				return 0, fmt.Errorf("media gallery item URL is required")
			}
		}
	case *discordgo.FileComponent:
		return 0, fmt.Errorf("file components are unavailable without a managed binary attachment")
	case *discordgo.Separator:
		if !topLevel || value.Spacing != nil && *value.Spacing != 1 && *value.Spacing != 2 {
			return 0, fmt.Errorf("separator spacing must be 1 or 2")
		}
	default:
		return 0, fmt.Errorf("unsupported component type %d", component.Type())
	}
	return 1, nil
}

func validateButton(button *discordgo.Button) error {
	if button.Style < 1 || button.Style > 6 || utf8.RuneCountInString(button.Label) > 80 {
		return fmt.Errorf("button style or label is invalid")
	}
	if strings.TrimSpace(button.Label) == "" && button.Emoji == nil && button.Style != discordgo.PremiumButton {
		return fmt.Errorf("button label or emoji is required")
	}
	if button.Emoji != nil {
		if button.Emoji.Name == "" && button.Emoji.ID == "" || button.Emoji.ID != "" && !snowflakePattern.MatchString(button.Emoji.ID) || utf8.RuneCountInString(button.Emoji.Name) > 32 {
			return fmt.Errorf("button emoji is invalid")
		}
	}
	if button.Style == discordgo.LinkButton {
		parsed, err := url.ParseRequestURI(button.URL)
		if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" || button.CustomID != "" {
			return fmt.Errorf("link buttons require url and forbid custom_id")
		}
	} else if button.Style == discordgo.PremiumButton {
		if !snowflakePattern.MatchString(button.SKUID) || button.CustomID != "" || button.URL != "" || button.Label != "" || button.Emoji != nil {
			return fmt.Errorf("premium buttons require sku_id only")
		}
	} else if button.CustomID == "" || len(button.CustomID) > 100 || button.URL != "" || button.SKUID != "" {
		return fmt.Errorf("interactive buttons require custom_id up to 100 characters")
	}
	return nil
}

func validateMentions(mentions AllowedMentions) error {
	if len(mentions.Roles) > 100 || len(mentions.Users) > 100 {
		return fmt.Errorf("explicit mention allowlists cannot exceed 100 IDs")
	}
	allowed, seen := map[string]bool{"everyone": true, "roles": true, "users": true}, map[string]bool{}
	for _, item := range mentions.Parse {
		if !allowed[item] || seen[item] {
			return fmt.Errorf("invalid allowed mention category %q", item)
		}
		seen[item] = true
	}
	if seen["roles"] && len(mentions.Roles) > 0 || seen["users"] && len(mentions.Users) > 0 {
		return fmt.Errorf("parsed and explicit mentions cannot overlap")
	}
	explicit := map[string]bool{}
	for _, id := range append(slices.Clone(mentions.Roles), mentions.Users...) {
		if !snowflakePattern.MatchString(id) {
			return fmt.Errorf("mention IDs must be Discord snowflake strings")
		}
		if explicit[id] {
			return fmt.Errorf("mention IDs must be unique")
		}
		explicit[id] = true
	}
	return nil
}

// DecodeComponent creates one canonical component from an SDK component.
func DecodeComponent(component discordgo.MessageComponent) (Component, error) {
	encoded, err := json.Marshal(component)
	return Component(encoded), err
}
