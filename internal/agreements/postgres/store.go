package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/agreements"
)

// Store persists business agreements.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates the PostgreSQL agreement store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Create inserts one unique agreement.
func (store *Store) Create(ctx context.Context, agreement agreements.Agreement) (agreements.Agreement, error) {
	err := store.pool.QueryRow(ctx, `INSERT INTO business_agreements
		(agreement_id, description, image_url, created_by) VALUES ($1, $2, NULLIF($3, ''), $4)
		ON CONFLICT DO NOTHING RETURNING created_at`, agreement.ID, agreement.Description,
		agreement.ImageURL, agreement.CreatedBy).Scan(&agreement.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return agreements.Agreement{}, agreements.ErrAlreadyExists
	}
	if err != nil {
		return agreements.Agreement{}, fmt.Errorf("create business agreement: %w", err)
	}
	return agreement, nil
}

// List returns all agreements ordered by identifier.
func (store *Store) List(ctx context.Context) ([]agreements.Agreement, error) {
	rows, err := store.pool.Query(ctx, `SELECT agreement_id, description, COALESCE(image_url, ''),
		created_by, created_at FROM business_agreements ORDER BY agreement_id`)
	if err != nil {
		return nil, fmt.Errorf("list business agreements: %w", err)
	}
	defer rows.Close()
	items := make([]agreements.Agreement, 0)
	for rows.Next() {
		var item agreements.Agreement
		if err := rows.Scan(&item.ID, &item.Description, &item.ImageURL, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan business agreement: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business agreements: %w", err)
	}
	return items, nil
}
