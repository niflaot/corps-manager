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
func (store *Store) Record(ctx context.Context, name string, userID string, displayName string,
	amount int64) (customers.Customer, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return customers.Customer{}, fmt.Errorf("begin customer visit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO frequent_customers (normalized_name, visits, total_spent, last_visit_at)
		VALUES ($1, 1, $2, now()) ON CONFLICT (normalized_name) DO UPDATE
		SET visits = frequent_customers.visits + 1,
			total_spent = frequent_customers.total_spent + EXCLUDED.total_spent,
			last_visit_at = now(), updated_at = now()`, name, amount)
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
	_, err = tx.Exec(ctx, `INSERT INTO frequent_customer_visits
		(customer_name, discord_user_id, display_name, amount) VALUES ($1, $2, $3, $4)`,
		name, userID, displayName, amount)
	if err != nil {
		return customers.Customer{}, fmt.Errorf("record customer visit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return customers.Customer{}, fmt.Errorf("commit customer visit: %w", err)
	}
	return store.Get(ctx, name)
}

// List returns all customers ordered by visit count and name.
func (store *Store) List(ctx context.Context) ([]customers.Customer, error) {
	return store.Search(ctx, customers.Query{Sort: customers.SortVisits})
}

// Search returns customers matching a normalized filter and period.
func (store *Store) Search(ctx context.Context, query customers.Query) ([]customers.Customer, error) {
	statement := allTimeQuery(customerOrder(query.Sort))
	arguments := []any{query.Name}
	if query.Days > 0 {
		statement = periodQuery(customerOrder(query.Sort))
		arguments = append(arguments, query.Days)
	}
	rows, err := store.pool.Query(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("search frequent customers: %w", err)
	}
	defer rows.Close()
	items := make([]customers.Customer, 0)
	for rows.Next() {
		var item customers.Customer
		if err := rows.Scan(&item.Name, &item.Visits, &item.TotalSpent, &item.CreatedAt,
			&item.UpdatedAt, &item.LastVisitAt, &item.AttendantCount); err != nil {
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
	err := store.pool.QueryRow(ctx, `SELECT c.normalized_name, c.visits, c.total_spent,
		c.created_at, c.updated_at, c.last_visit_at, COUNT(a.discord_user_id)
		FROM frequent_customers c LEFT JOIN frequent_customer_attendants a ON a.customer_name = c.normalized_name
		WHERE c.normalized_name = $1 GROUP BY c.normalized_name`, name).
		Scan(&item.Name, &item.Visits, &item.TotalSpent, &item.CreatedAt, &item.UpdatedAt,
			&item.LastVisitAt, &item.AttendantCount)
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

func customerOrder(sort customers.Sort) string {
	switch sort {
	case customers.SortName:
		return "name ASC"
	case customers.SortRecent:
		return "last_visit_at DESC, name ASC"
	case customers.SortVisits:
		return "visits DESC, total_spent DESC, name ASC"
	default:
		return "total_spent DESC, visits DESC, name ASC"
	}
}

func allTimeQuery(order string) string {
	return `SELECT c.normalized_name AS name, c.visits, c.total_spent, c.created_at,
		c.updated_at, c.last_visit_at, COUNT(a.discord_user_id)::int AS attendant_count
		FROM frequent_customers c LEFT JOIN frequent_customer_attendants a
		ON a.customer_name = c.normalized_name
		WHERE ($1 = '' OR c.normalized_name LIKE '%' || $1 || '%')
		GROUP BY c.normalized_name ORDER BY ` + order
}

func periodQuery(order string) string {
	return `SELECT v.customer_name AS name, COUNT(*)::bigint AS visits,
		COALESCE(SUM(v.amount), 0)::bigint AS total_spent, MIN(v.attended_at) AS created_at,
		MAX(v.attended_at) AS updated_at, MAX(v.attended_at) AS last_visit_at,
		COUNT(DISTINCT v.discord_user_id)::int AS attendant_count
		FROM frequent_customer_visits v
		WHERE ($1 = '' OR v.customer_name LIKE '%' || $1 || '%')
		AND v.attended_at >= now() - ($2 * interval '1 day')
		GROUP BY v.customer_name ORDER BY ` + order
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
