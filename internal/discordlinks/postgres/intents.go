package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

// CreateIntent creates or replays one idempotent OAuth attempt.
func (store *Store) CreateIntent(ctx context.Context, request discordlinks.CreateIntentRecord) (discordlinks.Intent, error) {
	intent, err := scanIntent(store.pool.QueryRow(ctx, `INSERT INTO discord_link_intents
		(kind,subject,completion_key,idempotency_key,request_hash,expires_at,created_at,updated_at)
		VALUES($1,NULLIF($2,''),$3,$4,$5,$6,$7,$7) ON CONFLICT(idempotency_key) DO NOTHING RETURNING `+intentColumns,
		request.Kind, request.Subject, request.CompletionKey, request.IdempotencyKey, request.RequestHash,
		request.ExpiresAt, request.Now))
	if err == nil {
		return intent, nil
	}
	if !errors.Is(err, discordlinks.ErrNotFound) {
		return discordlinks.Intent{}, err
	}
	var storedHash string
	row := store.pool.QueryRow(ctx, `SELECT `+intentColumns+`, request_hash FROM discord_link_intents WHERE idempotency_key=$1`,
		request.IdempotencyKey)
	if scanErr := row.Scan(&intent.ID, &intent.Kind, &intent.Subject, &intent.CompletionKey, &intent.Status,
		&intent.ExpiresAt, &intent.CreatedAt, &storedHash); scanErr != nil {
		return discordlinks.Intent{}, mapNotFound(scanErr)
	}
	if storedHash != request.RequestHash {
		return discordlinks.Intent{}, discordlinks.ErrConflict
	}
	return intent, nil
}

// StartIntent atomically binds a pending attempt to an OAuth state hash.
func (store *Store) StartIntent(ctx context.Context, intentID string, stateHash string, now time.Time) (discordlinks.Intent, error) {
	intent, err := scanIntent(store.pool.QueryRow(ctx, `UPDATE discord_link_intents
		SET status='started',state_hash=$2,started_at=$3,updated_at=$3
		WHERE id=$1 AND status='pending' AND expires_at>$3 RETURNING `+intentColumns, intentID, stateHash, now))
	if !errors.Is(err, discordlinks.ErrNotFound) {
		return intent, err
	}
	var expiresAt time.Time
	var status discordlinks.IntentStatus
	if lookupErr := store.pool.QueryRow(ctx, `SELECT status,expires_at FROM discord_link_intents WHERE id=$1`, intentID).
		Scan(&status, &expiresAt); lookupErr != nil {
		return discordlinks.Intent{}, mapNotFound(lookupErr)
	}
	if !expiresAt.After(now) {
		return discordlinks.Intent{}, discordlinks.ErrExpired
	}
	return discordlinks.Intent{}, discordlinks.ErrConflict
}

// ClaimIntentByState atomically owns one live callback by state hash.
func (store *Store) ClaimIntentByState(ctx context.Context, stateHash string, now time.Time) (discordlinks.Intent, error) {
	intent, err := scanIntent(store.pool.QueryRow(ctx, `UPDATE discord_link_intents
		SET status='processing',updated_at=$2 WHERE state_hash=$1 AND status='started' AND expires_at>$2
		RETURNING `+intentColumns, stateHash, now))
	if !errors.Is(err, discordlinks.ErrNotFound) {
		return intent, err
	}
	var expiresAt time.Time
	if lookupErr := store.pool.QueryRow(ctx, `SELECT expires_at FROM discord_link_intents WHERE state_hash=$1`, stateHash).
		Scan(&expiresAt); lookupErr != nil {
		return discordlinks.Intent{}, mapNotFound(lookupErr)
	}
	if !expiresAt.After(now) {
		return discordlinks.Intent{}, discordlinks.ErrExpired
	}
	return discordlinks.Intent{}, discordlinks.ErrConflict
}

// CompleteLinkIntent persists a proven identity and exchangeable link result.
func (store *Store) CompleteLinkIntent(ctx context.Context, request discordlinks.CompleteIntentRecord) (discordlinks.ResultStatus, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var subject string
	if err = tx.QueryRow(ctx, `SELECT subject FROM discord_link_intents
		WHERE id=$1 AND kind='link' AND status='processing' AND expires_at>$2 FOR UPDATE`, request.IntentID, request.Now).Scan(&subject); err != nil {
		return "", mapNotFound(err)
	}
	lockKeys := []string{"subject:" + subject, "discord:" + request.Identity.UserID}
	for _, key := range lockKeys {
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
			return "", err
		}
	}
	link, err := store.activeBySubject(ctx, tx, subject)
	if err == nil && link.DiscordUserID != request.Identity.UserID {
		return store.completeConflict(ctx, tx, request, discordlinks.ErrorCodeSubjectAlreadyLinked)
	}
	if err != nil && !errors.Is(err, discordlinks.ErrNotFound) {
		return "", err
	}
	other, otherErr := store.activeByDiscordUser(ctx, tx, request.Identity.UserID)
	if otherErr == nil && other.Subject != subject {
		return store.completeConflict(ctx, tx, request, discordlinks.ErrorCodeDiscordUserAlreadyLinked)
	}
	if otherErr != nil && !errors.Is(otherErr, discordlinks.ErrNotFound) {
		return "", otherErr
	}
	if err == nil {
		link, err = scanLink(tx.QueryRow(ctx, `UPDATE discord_links SET username=$2,global_name=$3,
			avatar_hash=$4,scopes=$5,updated_at=$6 WHERE id=$1 RETURNING `+linkColumns,
			link.ID, request.Identity.Username, request.Identity.GlobalName, request.Identity.AvatarHash,
			request.Scopes, request.Now))
	} else {
		link, err = scanLink(tx.QueryRow(ctx, `INSERT INTO discord_links
			(subject,discord_user_id,username,global_name,avatar_hash,scopes,linked_at,created_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$7,$7) RETURNING `+linkColumns,
			subject, request.Identity.UserID, request.Identity.Username, request.Identity.GlobalName,
			request.Identity.AvatarHash, request.Scopes, request.Now))
	}
	if err != nil {
		return "", err
	}
	if err = finishIntent(ctx, tx, request.IntentID, request.ResultHash, request.ResultExpiresAt,
		discordlinks.ResultStatusLinked, "", link.ID, request.Now); err != nil {
		return "", err
	}
	return discordlinks.ResultStatusLinked, tx.Commit(ctx)
}

func (store *Store) completeConflict(ctx context.Context, tx pgx.Tx, request discordlinks.CompleteIntentRecord,
	errorCode string) (discordlinks.ResultStatus, error) {
	if err := finishIntent(ctx, tx, request.IntentID, request.ResultHash, request.ResultExpiresAt,
		discordlinks.ResultStatusConflict, errorCode, "", request.Now); err != nil {
		return "", err
	}
	return discordlinks.ResultStatusConflict, tx.Commit(ctx)
}

// FailIntent persists an exchangeable safe failure result.
func (store *Store) FailIntent(ctx context.Context, request discordlinks.FailIntentRecord) error {
	command, err := store.pool.Exec(ctx, `UPDATE discord_link_intents SET status='completed',result_hash=$2,
		result_status=$3,result_error_code=$4,result_expires_at=$5,completed_at=$6,updated_at=$6
		WHERE id=$1 AND status='processing'`, request.IntentID, request.ResultHash, request.Status,
		request.ErrorCode, request.ResultExpiresAt, request.Now)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return discordlinks.ErrConflict
	}
	return nil
}

func finishIntent(ctx context.Context, tx pgx.Tx, intentID string, resultHash string, expiresAt time.Time,
	status discordlinks.ResultStatus, errorCode string, linkID string, now time.Time) error {
	var nullableLink any
	if linkID != "" {
		nullableLink = linkID
	}
	command, err := tx.Exec(ctx, `UPDATE discord_link_intents SET status='completed',result_hash=$2,
		result_status=$3,result_error_code=NULLIF($4,''),result_expires_at=$5,link_id=$6,
		completed_at=$7,updated_at=$7 WHERE id=$1 AND status='processing'`, intentID, resultHash,
		status, errorCode, expiresAt, nullableLink, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("%w: intent is not active", discordlinks.ErrConflict)
	}
	return nil
}
