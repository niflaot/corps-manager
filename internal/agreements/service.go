package agreements

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/niflaot/corps-manager/internal/messages"
)

const maximumCompanyNameLength = 80
const maximumCompanyIDLength = 60

var agreementIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var companyIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,59}$`)

// Service manages agreements and their managed Discord messages.
type Service struct {
	config     Config
	repository Repository
	messages   *messages.Service
	guildID    string
}

// NewService creates the agreements application service.
func NewService(config Config, repository Repository, messageService *messages.Service, guildID string) *Service {
	return &Service{config: config, repository: repository, messages: messageService, guildID: guildID}
}

// Create validates and persists one agreement.
func (service *Service) Create(ctx context.Context, companyID string, id string, description string, imageURL string,
	actor string) (Agreement, error) {
	if !service.config.Enabled {
		return Agreement{}, ErrDisabled
	}
	companyID = strings.TrimSpace(companyID)
	if !agreementIDPattern.MatchString(companyID) {
		return Agreement{}, ErrInvalidCompany
	}
	id = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), " ", "_"))
	description, imageURL, actor = strings.TrimSpace(description), strings.TrimSpace(imageURL), strings.TrimSpace(actor)
	if !agreementIDPattern.MatchString(id) {
		return Agreement{}, ErrInvalidID
	}
	if len([]rune(description)) < 3 || len([]rune(description)) > 1000 {
		return Agreement{}, ErrInvalidDescription
	}
	if imageURL != "" {
		parsed, err := url.ParseRequestURI(imageURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return Agreement{}, ErrInvalidImageURL
		}
	}
	if !snowflakePattern.MatchString(actor) {
		return Agreement{}, ErrInvalidID
	}
	agreement, err := service.repository.Create(ctx, Agreement{CompanyID: companyID, ID: id, Description: description,
		ImageURL: imageURL, CreatedBy: actor})
	if err != nil {
		return Agreement{}, err
	}
	return agreement, service.Publish(ctx)
}

// List returns every configured agreement.
func (service *Service) List(ctx context.Context) ([]Agreement, error) {
	if !service.config.Enabled {
		return nil, ErrDisabled
	}
	return service.repository.List(ctx)
}

// CreateCompany creates a company with its agreement channel and refreshes the panels.
func (service *Service) CreateCompany(ctx context.Context, id, name, channelID string) (Company, error) {
	company, err := validateCompany(id, name, channelID)
	if err != nil || !companyIDPattern.MatchString(company.ID) {
		return Company{}, ErrInvalidCompany
	}
	company, err = service.repository.CreateCompany(ctx, company)
	if err != nil {
		return Company{}, err
	}
	return company, service.refresh(ctx)
}

// UpdateCompany replaces a company's name and agreement channel and refreshes the panels.
func (service *Service) UpdateCompany(ctx context.Context, id, name, channelID string) (Company, error) {
	company, err := validateCompany(id, name, channelID)
	if err != nil {
		return Company{}, err
	}
	company, err = service.repository.UpdateCompany(ctx, company)
	if err != nil {
		return Company{}, err
	}
	return company, service.refresh(ctx)
}

// ListCompanies returns the configured companies independently of Discord publishing.
func (service *Service) ListCompanies(ctx context.Context) ([]Company, error) {
	return service.repository.ListCompanies(ctx)
}

// DeleteCompany removes an empty company, archives its list message and refreshes the panels.
func (service *Service) DeleteCompany(ctx context.Context, id string) error {
	if !agreementIDPattern.MatchString(id) {
		return ErrInvalidCompany
	}
	if err := service.repository.DeleteCompany(ctx, id); err != nil {
		return err
	}
	if !service.config.Enabled {
		return nil
	}
	return service.archive(ctx, companyMessageKey(id))
}

func (service *Service) refresh(ctx context.Context) error {
	if !service.config.Enabled {
		return nil
	}
	return service.Publish(ctx)
}

func validateCompany(id, name, channelID string) (Company, error) {
	company := Company{ID: strings.TrimSpace(id), Name: strings.TrimSpace(name), ChannelID: strings.TrimSpace(channelID)}
	valid := agreementIDPattern.MatchString(company.ID) && len([]rune(company.Name)) >= 2 &&
		len([]rune(company.Name)) <= maximumCompanyNameLength &&
		snowflakePattern.MatchString(company.ChannelID)
	if !valid {
		return Company{}, ErrInvalidCompany
	}
	return company, nil
}
