// Package agreements manages business agreements shown in Discord.
package agreements

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCompanyNotFound reports a missing company.
	ErrCompanyNotFound = errors.New("company not found")
	// ErrCompanyInUse prevents removing a company with agreements.
	ErrCompanyInUse = errors.New("company has agreements")
	// ErrInvalidCompany reports invalid company attributes.
	ErrInvalidCompany = errors.New("invalid company")
	// ErrAlreadyExists reports a duplicate agreement identifier.
	ErrAlreadyExists = errors.New("agreement already exists")
	// ErrInvalidID reports an invalid agreement identifier.
	ErrInvalidID = errors.New("invalid agreement id")
	// ErrInvalidDescription reports an invalid agreement description.
	ErrInvalidDescription = errors.New("invalid agreement description")
	// ErrInvalidImageURL reports an invalid optional image URL.
	ErrInvalidImageURL = errors.New("invalid agreement image URL")
	// ErrDisabled reports that agreements are disabled.
	ErrDisabled = errors.New("agreements are disabled")
)

// Agreement describes one business agreement.
type Agreement struct {
	// CompanyID identifies the company owning this agreement.
	CompanyID string `json:"companyId"`
	// CompanyName is the display name of the company.
	CompanyName string `json:"companyName"`
	// ID is the normalized business identifier.
	ID string `json:"id"`
	// Description explains the agreement.
	Description string `json:"description"`
	// ImageURL is an optional HTTPS illustration.
	ImageURL string `json:"imageUrl,omitempty"`
	// CreatedBy is the Discord user snowflake that added it.
	CreatedBy string `json:"createdBy"`
	// CreatedAt is the persistence creation time.
	CreatedAt time.Time `json:"createdAt"`
}

// Company describes one configurable agreement company.
type Company struct {
	// ID is the stable company identifier.
	ID string `json:"id"`
	// Name is the display name shown in Discord forms.
	Name string `json:"name"`
}

// Repository persists business agreements.
type Repository interface {
	// CreateCompany creates a uniquely identified company.
	CreateCompany(context.Context, Company) (Company, error)
	// ListCompanies returns all companies ordered by ID.
	ListCompanies(context.Context) ([]Company, error)
	// DeleteCompany deletes a company only when it has no agreements.
	DeleteCompany(context.Context, string) error
	// Create inserts one unique agreement.
	Create(context.Context, Agreement) (Agreement, error)
	// List returns agreements ordered by identifier.
	List(context.Context) ([]Agreement, error)
}
