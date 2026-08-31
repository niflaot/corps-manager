package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/customers"
)

// Store persists frequent customers and attendants.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates the PostgreSQL customer store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Record atomically increments a customer and its unique attendant.
func (store *Store) Record(ctx context.Context, name string, userID string, displayName string) (customers.Customer, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return customers.Customer{}, fmt.Errorf("begin customer visit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO frequent_customers (normalized_name, visits)
		VALUES ($1, 1) ON CONFLICT (normalized_name) DO UPDATE
		SET visits = frequent_customers.visits + 1, updated_at = now()`, name)
	if err != nil {
		return customers.Customer{}, fmt.Errorf("record customer visit: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO frequent_customer_attendants
		(customer_name, discord_user_id, display_name, visits) VALUES ($1, $2, $3, 1)
		ON CONFLICT (customer_name, discord_user_id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
			visits = frequent_customer_attendants.visits + 1, last_attended_at = now()`, name, userID, displayName)
	if err != nil {
		return customers.Customer{}, fmt.Errorf("record customer attendant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return customers.Customer{}, fmt.Errorf("commit customer visit: %w", err)
	}
	return store.Get(ctx, name)
}

// List returns all customers ordered by visit count and name.
func (store *Store) List(ctx context.Context) ([]customers.Customer, error) {
	rows, err := store.pool.Query(ctx, `SELECT normalized_name, visits, created_at, updated_at
		FROM frequent_customers ORDER BY visits DESC, normalized_name`)
	if err != nil {
		return nil, fmt.Errorf("list frequent customers: %w", err)
	}
	defer rows.Close()
	items := make([]customers.Customer, 0)
	for rows.Next() {
		var item customers.Customer
		if err := rows.Scan(&item.Name, &item.Visits, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan frequent customer: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate frequent customers: %w", err)
	}
	return items, nil
}

// Get returns one customer with unique attendants.
func (store *Store) Get(ctx context.Context, name string) (customers.Customer, error) {
	var item customers.Customer
	err := store.pool.QueryRow(ctx, `SELECT normalized_name, visits, created_at, updated_at
		FROM frequent_customers WHERE normalized_name = $1`, name).
		Scan(&item.Name, &item.Visits, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return customers.Customer{}, customers.ErrNotFound
	}
	if err != nil {
		return customers.Customer{}, fmt.Errorf("get frequent customer: %w", err)
	}
	rows, err := store.pool.Query(ctx, `SELECT discord_user_id, display_name, visits,
		first_attended_at, last_attended_at FROM frequent_customer_attendants
		WHERE customer_name = $1 ORDER BY visits DESC, display_name`, name)
	if err != nil {
		return customers.Customer{}, fmt.Errorf("list customer attendants: %w", err)
	}
	defer rows.Close()
	item.Attendants = make([]customers.Attendant, 0)
	for rows.Next() {
		var attendant customers.Attendant
		if err := rows.Scan(&attendant.DiscordUserID, &attendant.DisplayName, &attendant.Visits,
			&attendant.FirstAttendedAt, &attendant.LastAttendedAt); err != nil {
			return customers.Customer{}, fmt.Errorf("scan customer attendant: %w", err)
		}
		item.Attendants = append(item.Attendants, attendant)
	}
	if err := rows.Err(); err != nil {
		return customers.Customer{}, fmt.Errorf("iterate customer attendants: %w", err)
	}
	return item, nil
}

// Delete removes one customer and cascading attendant history.
func (store *Store) Delete(ctx context.Context, name string) error {
	result, err := store.pool.Exec(ctx, `DELETE FROM frequent_customers WHERE normalized_name = $1`, name)
	if err != nil {
		return fmt.Errorf("delete frequent customer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return customers.ErrNotFound
	}
	return nil
}
