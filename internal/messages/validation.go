package messages

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

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

// Validate verifies Discord message limits and mention policy.
func (payload Payload) Validate() error {
	payload = payload.Normalize()
	if utf8.RuneCountInString(payload.Content) > 2000 {
		return fmt.Errorf("content exceeds 2000 characters")
	}
	if strings.TrimSpace(payload.Content) == "" && len(payload.Embeds) == 0 {
		return fmt.Errorf("content or at least one embed is required")
	}
	if len(payload.Embeds) > 10 {
		return fmt.Errorf("embeds exceeds 10 items")
	}
	total := 0
	for index, embed := range payload.Embeds {
		count, err := validateEmbed(embed)
		if err != nil {
			return fmt.Errorf("embed %d: %w", index, err)
		}
		total += count
	}
	if total > 6000 {
		return fmt.Errorf("combined embed text exceeds 6000 characters")
	}
	return validateMentions(payload.AllowedMentions)
}

func validateEmbed(embed Embed) (int, error) {
	type constrained struct {
		name  string
		value string
		limit int
	}
	lengths := []constrained{{"title", embed.Title, 256}, {"description", embed.Description, 4096}}
	if embed.Footer != nil {
		if strings.TrimSpace(embed.Footer.Text) == "" {
			return 0, fmt.Errorf("footer.text is required")
		}
		lengths = append(lengths, constrained{"footer.text", embed.Footer.Text, 2048})
		if err := validateURL(embed.Footer.IconURL); err != nil {
			return 0, fmt.Errorf("footer.iconUrl: %w", err)
		}
	}
	if embed.Author != nil {
		if strings.TrimSpace(embed.Author.Name) == "" {
			return 0, fmt.Errorf("author.name is required")
		}
		lengths = append(lengths, constrained{"author.name", embed.Author.Name, 256})
		if err := validateURL(embed.Author.URL); err != nil {
			return 0, fmt.Errorf("author.url: %w", err)
		}
		if err := validateURL(embed.Author.IconURL); err != nil {
			return 0, fmt.Errorf("author.iconUrl: %w", err)
		}
	}
	if len(embed.Fields) > 25 {
		return 0, fmt.Errorf("fields exceeds 25 items")
	}
	for _, field := range embed.Fields {
		if strings.TrimSpace(field.Name) == "" || strings.TrimSpace(field.Value) == "" {
			return 0, fmt.Errorf("field name and value are required")
		}
		lengths = append(lengths, constrained{"field.name", field.Name, 256}, constrained{"field.value", field.Value, 1024})
	}
	total := 0
	for _, item := range lengths {
		count := utf8.RuneCountInString(item.value)
		if count > item.limit {
			return 0, fmt.Errorf("%s exceeds %d characters", item.name, item.limit)
		}
		total += count
	}
	if embed.Color < 0 || embed.Color > 0xffffff {
		return 0, fmt.Errorf("color must be between 0 and 16777215")
	}
	if embed.Timestamp != "" {
		if _, err := time.Parse(time.RFC3339, embed.Timestamp); err != nil {
			return 0, fmt.Errorf("timestamp must be RFC3339")
		}
	}
	if (embed.Image != nil && embed.Image.URL == "") || (embed.Thumbnail != nil && embed.Thumbnail.URL == "") {
		return 0, fmt.Errorf("embed media URL is required")
	}
	for name, raw := range map[string]string{"url": embed.URL, "image.url": mediaURL(embed.Image), "thumbnail.url": mediaURL(embed.Thumbnail)} {
		if err := validateURL(raw); err != nil {
			return 0, fmt.Errorf("%s: %w", name, err)
		}
	}
	return total, nil
}

func validateMentions(mentions AllowedMentions) error {
	if len(mentions.Roles) > 100 || len(mentions.Users) > 100 {
		return fmt.Errorf("explicit mention allowlists cannot exceed 100 IDs")
	}
	allowed := map[string]bool{"everyone": true, "roles": true, "users": true}
	seen := make(map[string]bool, len(mentions.Parse))
	for _, item := range mentions.Parse {
		if !allowed[item] || seen[item] {
			return fmt.Errorf("invalid allowed mention category %q", item)
		}
		seen[item] = true
	}
	if seen["roles"] && len(mentions.Roles) > 0 || seen["users"] && len(mentions.Users) > 0 {
		return fmt.Errorf("parsed and explicit mentions cannot overlap")
	}
	explicit := make(map[string]bool, len(mentions.Roles)+len(mentions.Users))
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

func validateURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("must be an absolute HTTP(S) URL")
	}
	return nil
}

func mediaURL(media *EmbedMedia) string {
	if media == nil {
		return ""
	}
	return media.URL
}
