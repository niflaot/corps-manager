package announcements

import (
	"github.com/niflaot/corps-manager/platform/clock"
	"go.uber.org/fx"
)

// Module provides business-opening announcement use cases.
var Module = fx.Module("announcements", fx.Provide(
	LoadConfig,
	NewService,
))

// Service coordinates cooldown persistence and announcement publication.
type Service struct {
	config     Config
	repository Repository
	gateway    Gateway
	clock      clock.Clock
}

// NewService creates the business-opening announcement service.
func NewService(config Config, repository Repository, gateway Gateway, serviceClock clock.Clock) *Service {
	return &Service{config: config, repository: repository, gateway: gateway, clock: serviceClock}
}
