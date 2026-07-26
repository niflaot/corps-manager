package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/pixelados-net/discord-bot/internal/verification/notification"
)

// ClaimDue leases one bounded due batch without holding a network transaction.
func (store *Store) ClaimDue(ctx context.Context, request notification.ClaimRequest) ([]notification.Delivery, error) {
	rows, err := store.pool.Query(ctx, `WITH due AS (
		SELECT id AS due_id FROM verification_notification_outbox
		WHERE ((state IN ('pending','retry') AND next_attempt_at <= $1)
			OR (state = 'delivering' AND lease_until <= $1))
		ORDER BY next_attempt_at,id LIMIT $2 FOR UPDATE SKIP LOCKED
	) UPDATE verification_notification_outbox AS delivery
		SET state='delivering',lease_owner=$3,lease_until=$1+make_interval(secs=>$4),updated_at=$1
		FROM due WHERE delivery.id=due.due_id RETURNING `+deliveryColumns,
		request.Now, request.Limit, request.Owner, int64(request.LeaseDuration/time.Second))
	if err != nil {
		return nil, fmt.Errorf("claim verification notifications: %w", err)
	}
	defer rows.Close()
	deliveries := make([]notification.Delivery, 0, request.Limit)
	for rows.Next() {
		delivery, scanErr := scanDelivery(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, rows.Err()
}

// Complete marks one leased delivery successful.
func (store *Store) Complete(ctx context.Context, completion notification.Completion) error {
	command, err := store.pool.Exec(ctx, `UPDATE verification_notification_outbox
		SET state='delivered',attempts=attempts+1,discord_message_id=$3,delivered_at=$4,
			lease_owner=NULL,lease_until=NULL,last_error=NULL,updated_at=$4
		WHERE id=$1 AND lease_owner=$2`,
		completion.ID, completion.Owner, completion.DiscordMessageID, completion.DeliveredAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return notification.ErrConflict
	}
	return nil
}

// Release schedules retry or moves one leased delivery to the dead-letter state.
func (store *Store) Release(ctx context.Context, release notification.Release) error {
	command, err := store.pool.Exec(ctx, `UPDATE verification_notification_outbox
		SET state=$3,attempts=attempts+1,next_attempt_at=$4,last_error=$5,
			lease_owner=NULL,lease_until=NULL,updated_at=now()
		WHERE id=$1 AND lease_owner=$2`,
		release.ID, release.Owner, release.State, release.NextAttemptAt, release.Error)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return notification.ErrConflict
	}
	return nil
}
