// Package postgres persists the durable verification notification outbox.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/verification/notification"
)

const deliveryColumns = `id::text,idempotency_key,kind,user_id,group_id,group_key,state,attempts`

// Store is the PostgreSQL verification notification repository.
type Store struct {
	// pool owns PostgreSQL query execution.
	pool *pgxpool.Pool
}

// NewStore creates a PostgreSQL verification notification repository.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Enqueue inserts one event unless its idempotency key already exists.
func (store *Store) Enqueue(ctx context.Context, event notification.Event) (bool, error) {
	command, err := store.pool.Exec(ctx, `INSERT INTO verification_notification_outbox
		(idempotency_key,kind,user_id,group_id,group_key) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(idempotency_key) DO NOTHING`,
		event.IdempotencyKey, event.Kind, event.UserID, event.GroupID, event.GroupKey)
	if err != nil {
		return false, err
	}
	return command.RowsAffected() == 1, nil
}

// scanner abstracts PostgreSQL rows and row values.
type scanner interface{ Scan(...any) error }

// scanDelivery decodes one outbox row.
func scanDelivery(row scanner) (notification.Delivery, error) {
	var delivery notification.Delivery
	if err := row.Scan(&delivery.ID, &delivery.IdempotencyKey, &delivery.Kind, &delivery.UserID,
		&delivery.GroupID, &delivery.GroupKey, &delivery.State, &delivery.Attempts); err != nil {
		return notification.Delivery{}, mapError(err)
	}
	return delivery, nil
}

// mapError converts PostgreSQL conflicts to domain errors.
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.ErrConflict
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return notification.ErrConflict
	}
	return err
}
