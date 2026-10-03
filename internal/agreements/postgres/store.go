package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/agreements"
)

const uniqueViolation = "23505"
const foreignKeyViolation = "23503"
const restrictViolation = "23001"

// Store persists companies and their agreements.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates the PostgreSQL agreement store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Create inserts an agreement with an identifier unique within its company.
func (store *Store) Create(ctx context.Context, item agreements.Agreement) (agreements.Agreement, error) {
	err := store.pool.QueryRow(ctx, `INSERT INTO business_agreements
 (company_id, agreement_id, description, image_url, created_by) VALUES ($1,$2,$3,NULLIF($4,''),$5)
 RETURNING created_at, (SELECT name FROM agreement_companies WHERE company_id = $1)`,
		item.CompanyID, item.ID, item.Description, item.ImageURL, item.CreatedBy).Scan(&item.CreatedAt, &item.CompanyName)
	if err != nil {
		return agreements.Agreement{}, mapError(err)
	}
	return item, nil
}

// List returns agreements grouped by company and identifier.
func (store *Store) List(ctx context.Context) ([]agreements.Agreement, error) {
	rows, err := store.pool.Query(ctx, `SELECT a.company_id, c.name, a.agreement_id, a.description,
 COALESCE(a.image_url,''), a.created_by, a.created_at FROM business_agreements a
 JOIN agreement_companies c USING (company_id) ORDER BY a.company_id, a.agreement_id`)
	if err != nil {
		return nil, fmt.Errorf("list agreements: %w", err)
	}
	defer rows.Close()
	items := make([]agreements.Agreement, 0)
	for rows.Next() {
		var item agreements.Agreement
		if err := rows.Scan(&item.CompanyID, &item.CompanyName, &item.ID, &item.Description, &item.ImageURL, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CreateCompany inserts a uniquely identified company.
func (store *Store) CreateCompany(ctx context.Context, company agreements.Company) (agreements.Company, error) {
	_, err := store.pool.Exec(ctx, `INSERT INTO agreement_companies (company_id,name,channel_id) VALUES ($1,$2,NULLIF($3,''))`,
		company.ID, company.Name, company.ChannelID)
	return company, mapError(err)
}

// UpdateCompany replaces the name and channel of an existing company.
func (store *Store) UpdateCompany(ctx context.Context, company agreements.Company) (agreements.Company, error) {
	result, err := store.pool.Exec(ctx, `UPDATE agreement_companies SET name = $2, channel_id = NULLIF($3,'') WHERE company_id = $1`,
		company.ID, company.Name, company.ChannelID)
	if err != nil {
		return agreements.Company{}, mapError(err)
	}
	if result.RowsAffected() == 0 {
		return agreements.Company{}, agreements.ErrCompanyNotFound
	}
	return company, nil
}

// ListCompanies returns all companies ordered by identifier.
func (store *Store) ListCompanies(ctx context.Context) ([]agreements.Company, error) {
	rows, err := store.pool.Query(ctx, `SELECT company_id,name,COALESCE(channel_id,'') FROM agreement_companies ORDER BY company_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]agreements.Company, 0)
	for rows.Next() {
		var item agreements.Company
		if err := rows.Scan(&item.ID, &item.Name, &item.ChannelID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// DeleteCompany deletes only companies without agreements, enforced by the foreign key.
func (store *Store) DeleteCompany(ctx context.Context, id string) error {
	result, err := store.pool.Exec(ctx, `DELETE FROM agreement_companies WHERE company_id = $1`, id)
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && (postgresError.Code == foreignKeyViolation || postgresError.Code == restrictViolation) {
		return agreements.ErrCompanyInUse
	}
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return agreements.ErrCompanyNotFound
	}
	return nil
}

func mapError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case uniqueViolation:
			return agreements.ErrAlreadyExists
		case foreignKeyViolation:
			return agreements.ErrCompanyNotFound
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return agreements.ErrCompanyNotFound
	}
	return err
}
