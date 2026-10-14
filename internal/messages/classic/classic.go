// Package classic converts classic Discord messages with embeds into Components V2 payloads.
package classic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/niflaot/corps-manager/internal/messages"
)

// maxTextLength is the Components V2 text display limit.
const maxTextLength = 4000

// containerType, sectionType, textType, galleryType, and thumbnailType are Components V2 identifiers.
const (
	textType      = 10
	sectionType   = 9
	containerType = 17
	galleryType   = 12
	thumbnailType = 11
)

// ignored lists classic message fields with no Components V2 equivalent.
var ignored = map[string]bool{"username": true, "avatar_url": true, "tts": true, "flags": true, "nonce": true,
	"attachments": true, "thread_name": true, "sticker_ids": true, "poll": true, "message_reference": true}

type media struct {
	URL string `json:"url"`
}

type embed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Color       int    `json:"color"`
	Timestamp   string `json:"timestamp"`
	Footer      *struct {
		Text string `json:"text"`
	} `json:"footer"`
	Image     *media `json:"image"`
	Thumbnail *media `json:"thumbnail"`
	Author    *struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"author"`
	Fields []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"fields"`
}

type message struct {
	Content         string               `json:"content"`
	Embeds          []embed              `json:"embeds"`
	Components      []messages.Component `json:"components"`
	AllowedMentions *struct {
		Parse []string `json:"parse"`
		Roles []string `json:"roles"`
		Users []string `json:"users"`
	} `json:"allowed_mentions"`
}

// IsClassic reports whether raw is a classic message object carrying content or embeds.
func IsClassic(raw []byte) bool {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(raw), &keys); err != nil {
		return false
	}
	_, content := keys["content"]
	_, embeds := keys["embeds"]
	return content || embeds
}

// Convert transforms a classic message into a Components V2 payload; callers must validate the result.
func Convert(raw []byte) (messages.Payload, error) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return messages.Payload{}, fmt.Errorf("JSON inválido: %w", err)
	}
	for key := range keys {
		switch key {
		case "content", "embeds", "components", "allowed_mentions":
		default:
			if !ignored[key] {
				return messages.Payload{}, fmt.Errorf("el campo %q no se puede convertir a Components V2", key)
			}
		}
	}
	var source message
	if err := json.Unmarshal(raw, &source); err != nil {
		return messages.Payload{}, fmt.Errorf("JSON inválido: %w", err)
	}
	var payload messages.Payload
	if strings.TrimSpace(source.Content) != "" {
		if err := checkLength("content", source.Content); err != nil {
			return messages.Payload{}, err
		}
		payload.Components = append(payload.Components, encode(map[string]any{"type": textType, "content": source.Content}))
	}
	for index, item := range source.Embeds {
		container, err := convertEmbed(item)
		if err != nil {
			return messages.Payload{}, fmt.Errorf("embed %d: %w", index+1, err)
		}
		if container != nil {
			payload.Components = append(payload.Components, encode(container))
		}
	}
	payload.Components = append(payload.Components, source.Components...)
	if len(payload.Components) == 0 {
		return messages.Payload{}, errors.New("el mensaje no tiene contenido ni embeds")
	}
	payload.AllowedMentions = messages.AllowedMentions{Parse: []string{}}
	if mentions := source.AllowedMentions; mentions != nil {
		payload.AllowedMentions = messages.AllowedMentions{Parse: mentions.Parse, Roles: mentions.Roles, Users: mentions.Users}
	}
	return payload, nil
}

func convertEmbed(item embed) (map[string]any, error) {
	var texts []map[string]any
	for _, block := range []string{header(item), item.Description, fieldsText(item)} {
		if strings.TrimSpace(block) == "" {
			continue
		}
		if err := checkLength("texto", block); err != nil {
			return nil, err
		}
		texts = append(texts, map[string]any{"type": textType, "content": block})
	}
	var children []any
	switch {
	case item.Thumbnail != nil && item.Thumbnail.URL != "" && len(texts) > 0:
		children = append(children, map[string]any{"type": sectionType, "components": texts,
			"accessory": map[string]any{"type": thumbnailType, "media": map[string]any{"url": item.Thumbnail.URL}}})
	default:
		for _, text := range texts {
			children = append(children, text)
		}
		if item.Thumbnail != nil && item.Thumbnail.URL != "" {
			children = append(children, gallery(item.Thumbnail.URL))
		}
	}
	if item.Image != nil && item.Image.URL != "" {
		children = append(children, gallery(item.Image.URL))
	}
	if footer := footerText(item); footer != "" {
		children = append(children, map[string]any{"type": textType, "content": footer})
	}
	if len(children) == 0 {
		return nil, nil
	}
	container := map[string]any{"type": containerType, "components": children}
	if item.Color > 0 && item.Color <= 0xffffff {
		container["accent_color"] = item.Color
	}
	return container, nil
}

func header(item embed) string {
	title := strings.TrimSpace(item.Title)
	if title != "" {
		if item.URL != "" {
			title = "[" + title + "](" + item.URL + ")"
		}
		title = "## " + title
	}
	author := ""
	if item.Author != nil && strings.TrimSpace(item.Author.Name) != "" {
		author = strings.TrimSpace(item.Author.Name)
		if item.Author.URL != "" {
			author = "[" + author + "](" + item.Author.URL + ")"
		}
		if title == "" {
			author = "### " + author
		} else {
			author = "-# " + author
		}
	}
	return strings.TrimSpace(author + "\n" + title)
}

func fieldsText(item embed) string {
	parts := make([]string, 0, len(item.Fields))
	for _, field := range item.Fields {
		parts = append(parts, "**"+field.Name+"**\n"+field.Value)
	}
	return strings.Join(parts, "\n\n")
}

func footerText(item embed) string {
	parts := []string{}
	if item.Footer != nil && strings.TrimSpace(item.Footer.Text) != "" {
		parts = append(parts, strings.TrimSpace(item.Footer.Text))
	}
	if stamp, err := time.Parse(time.RFC3339, item.Timestamp); err == nil {
		parts = append(parts, fmt.Sprintf("<t:%d:f>", stamp.Unix()))
	}
	if len(parts) == 0 {
		return ""
	}
	return "-# " + strings.Join(parts, " • ")
}

func gallery(url string) map[string]any {
	return map[string]any{"type": galleryType, "items": []any{map[string]any{"media": map[string]any{"url": url}}}}
}

func checkLength(name, text string) error {
	if utf8.RuneCountInString(text) > maxTextLength {
		return fmt.Errorf("el %s supera %d caracteres, el máximo de Components V2", name, maxTextLength)
	}
	return nil
}

func encode(value map[string]any) messages.Component {
	encoded, _ := json.Marshal(value)
	return messages.Component(encoded)
}
