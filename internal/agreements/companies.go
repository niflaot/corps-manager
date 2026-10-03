package agreements

import (
	"context"
	"strings"
)

const maximumCompanyNameLength = 80

// CreateCompany creates a company available to the agreement form.
func (service *Service) CreateCompany(ctx context.Context, id, name string) (Company, error) {
	id, name = strings.TrimSpace(id), strings.TrimSpace(name)
	if !agreementIDPattern.MatchString(id) || len([]rune(name)) < 2 || len([]rune(name)) > maximumCompanyNameLength {
		return Company{}, ErrInvalidCompany
	}
	return service.repository.CreateCompany(ctx, Company{ID: id, Name: name})
}

// ListCompanies returns the configured companies independently of Discord publishing.
func (service *Service) ListCompanies(ctx context.Context) ([]Company, error) {
	return service.repository.ListCompanies(ctx)
}

// DeleteCompany removes an empty company; existing agreements prevent deletion.
func (service *Service) DeleteCompany(ctx context.Context, id string) error {
	if !agreementIDPattern.MatchString(id) {
		return ErrInvalidCompany
	}
	return service.repository.DeleteCompany(ctx, id)
}
