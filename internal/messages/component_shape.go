package messages

import (
	"encoding/json"
	"fmt"
)

var componentFields = map[int]map[string]bool{
	1:  fields("type", "id", "components"),
	2:  fields("type", "id", "label", "style", "disabled", "emoji", "url", "custom_id", "sku_id"),
	9:  fields("type", "id", "components", "accessory"),
	10: fields("type", "content"),
	11: fields("type", "id", "media", "description", "spoiler"),
	12: fields("type", "id", "items"),
	13: fields("type", "id", "file", "spoiler"),
	14: fields("type", "id", "divider", "spacing"),
	17: fields("type", "id", "accent_color", "spoiler", "components"),
}

func validateComponentJSON(raw []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return fmt.Errorf("component must be a JSON object")
	}
	var componentType int
	if err := json.Unmarshal(object["type"], &componentType); err != nil {
		return fmt.Errorf("integer type is required")
	}
	allowed, ok := componentFields[componentType]
	if !ok {
		return fmt.Errorf("unsupported component type %d", componentType)
	}
	for key := range object {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q for component type %d", key, componentType)
		}
	}
	for _, key := range []string{"components"} {
		if children, exists := object[key]; exists {
			var values []json.RawMessage
			if err := json.Unmarshal(children, &values); err != nil {
				return fmt.Errorf("%s must be an array", key)
			}
			for _, child := range values {
				if err := validateComponentJSON(child); err != nil {
					return err
				}
			}
		}
	}
	if accessory, exists := object["accessory"]; exists {
		if err := validateComponentJSON(accessory); err != nil {
			return err
		}
	}
	if emoji, exists := object["emoji"]; exists {
		if err := validateObjectFields(emoji, fields("name", "id", "animated")); err != nil {
			return fmt.Errorf("emoji: %w", err)
		}
	}
	if media, exists := object["media"]; exists {
		if err := validateObjectFields(media, fields("url")); err != nil {
			return fmt.Errorf("media: %w", err)
		}
	}
	if file, exists := object["file"]; exists {
		if err := validateObjectFields(file, fields("url")); err != nil {
			return fmt.Errorf("file: %w", err)
		}
	}
	if items, exists := object["items"]; exists {
		var values []map[string]json.RawMessage
		if err := json.Unmarshal(items, &values); err != nil {
			return fmt.Errorf("items must be an array")
		}
		for _, item := range values {
			for key := range item {
				if !fields("media", "description", "spoiler")[key] {
					return fmt.Errorf("unknown media item field %q", key)
				}
			}
			if err := validateObjectFields(item["media"], fields("url")); err != nil {
				return fmt.Errorf("media item: %w", err)
			}
		}
	}
	return nil
}

func validateObjectFields(raw []byte, allowed map[string]bool) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return fmt.Errorf("must be an object")
	}
	for key := range object {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func fields(names ...string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result
}
