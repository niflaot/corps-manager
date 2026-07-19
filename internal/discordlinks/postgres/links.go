package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

// ExchangeResult consumes or idempotently replays one result.
func (store *Store) ExchangeResult(ctx context.Context, resultHash string, consumerKey string,
	now time.Time) (discordlinks.Result, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return discordlinks.Result{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var result discordlinks.Result
	var linkID *string
	var expiresAt time.Time
	var consumedAt *time.Time
	var storedConsumer *string
	err = tx.QueryRow(ctx, `SELECT result_status,COALESCE(subject,''),COALESCE(result_error_code,''),link_id,result_expires_at,
		result_consumed_at,result_consumer_key FROM discord_link_intents
		WHERE result_hash=$1 AND status='completed' FOR UPDATE`, resultHash).
		Scan(&result.Status, &result.Subject, &result.ErrorCode, &linkID, &expiresAt, &consumedAt, &storedConsumer)
	if err != nil {
		return discordlinks.Result{}, mapNotFound(err)
	}
	if !expiresAt.After(now) {
		return discordlinks.Result{}, discordlinks.ErrExpired
	}
	if consumedAt != nil && (storedConsumer == nil || *storedConsumer != consumerKey) {
		return discordlinks.Result{}, discordlinks.ErrGone
	}
	if consumedAt == nil {
		if _, err = tx.Exec(ctx, `UPDATE discord_link_intents SET result_consumed_at=$2,
			result_consumer_key=$3,updated_at=$2 WHERE result_hash=$1`, resultHash, now, consumerKey); err != nil {
			return discordlinks.Result{}, err
		}
	}
	if linkID != nil {
		link, linkErr := scanLink(tx.QueryRow(ctx, `SELECT `+linkColumns+` FROM discord_links WHERE id=$1`, *linkID))
		if linkErr != nil {
			return discordlinks.Result{}, linkErr
		}
		result.Link = &link
		if result.Subject == "" {
			result.Subject = link.Subject
		}
	}
	return result, tx.Commit(ctx)
}

// LinkBySubject returns the latest link history for one local subject.
func (store *Store) LinkBySubject(ctx context.Context, subject string) (discordlinks.Link, error) {
	return scanLink(store.pool.QueryRow(ctx, `SELECT `+linkColumns+` FROM discord_links
		WHERE subject=$1 ORDER BY created_at DESC LIMIT 1`, subject))
}

// LinkByDiscordUser returns the active link for one Discord user.
func (store *Store) LinkByDiscordUser(ctx context.Context, userID string) (discordlinks.Link, error) {
	return scanLink(store.pool.QueryRow(ctx, `SELECT `+linkColumns+` FROM discord_links
		WHERE discord_user_id=$1 AND unlinked_at IS NULL`, userID))
}

// Unlink conditionally removes one active association.
func (store *Store) Unlink(ctx context.Context, subject string, now time.Time) (discordlinks.Link, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return discordlinks.Link{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "subject:"+subject); err != nil {
		return discordlinks.Link{}, err
	}
	active, err := store.activeBySubject(ctx, tx, subject)
	if errors.Is(err, discordlinks.ErrNotFound) {
		link, historyErr := scanLink(tx.QueryRow(ctx, `SELECT `+linkColumns+` FROM discord_links
			WHERE subject=$1 ORDER BY created_at DESC LIMIT 1`, subject))
		if historyErr != nil {
			return discordlinks.Link{}, historyErr
		}
		return link, tx.Commit(ctx)
	}
	if err != nil {
		return discordlinks.Link{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		"discord:"+active.DiscordUserID); err != nil {
		return discordlinks.Link{}, err
	}
	link, err := scanLink(tx.QueryRow(ctx, `UPDATE discord_links SET unlinked_at=$2,updated_at=$2
		WHERE id=$1 AND unlinked_at IS NULL RETURNING `+linkColumns, active.ID, now))
	if err != nil {
		return discordlinks.Link{}, err
	}
	return link, tx.Commit(ctx)
}

func (store *Store) activeBySubject(ctx context.Context, tx pgx.Tx, subject string) (discordlinks.Link, error) {
	return scanLink(tx.QueryRow(ctx, `SELECT `+linkColumns+` FROM discord_links
		WHERE subject=$1 AND unlinked_at IS NULL`, subject))
}

func (store *Store) activeByDiscordUser(ctx context.Context, tx pgx.Tx, userID string) (discordlinks.Link, error) {
	return scanLink(tx.QueryRow(ctx, `SELECT `+linkColumns+` FROM discord_links
		WHERE discord_user_id=$1 AND unlinked_at IS NULL`, userID))
}

// DeleteExpiredIntents removes OAuth artifacts older than the retention boundary.
func (store *Store) DeleteExpiredIntents(ctx context.Context, before time.Time) (int64, error) {
	command, err := store.pool.Exec(ctx, `DELETE FROM discord_link_intents WHERE updated_at<$1`, before)
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}
