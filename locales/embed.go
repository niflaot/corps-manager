// Package locales embeds the repository's default localization catalog.
package locales

import _ "embed"

// Default contains the default messages.json bytes.
//
//go:embed messages.json
var Default []byte
