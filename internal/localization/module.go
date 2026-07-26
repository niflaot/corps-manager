package localization

import (
	"context"

	"go.uber.org/fx"
)

// Module provides localization configuration and the immutable catalog.
var Module = fx.Module("localization", fx.Provide(LoadConfig, provideCatalog))

// provideCatalog loads the immutable startup catalog.
func provideCatalog(config Config) (*Catalog, error) {
	return Load(context.Background(), config)
}
