package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

// CompleteLoginIntent resolves a proven identity to an existing active link.
func (store *Store) CompleteLoginIntent(ctx context.Context,
	request discordlinks.CompleteLoginIntentRecord) (discordlinks.ResultStatus, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var intentID string
	if err = tx.QueryRow(ctx, `SELECT id FROM discord_link_intents
		WHERE id=$1 AND kind='login' AND status='processing' AND expires_at>$2 FOR UPDATE`,
		request.IntentID, request.Now).Scan(&intentID); err != nil {
		return "", mapNotFound(err)
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		"discord:"+request.Identity.UserID); err != nil {
		return "", err
	}
	link, err := store.activeByDiscordUser(ctx, tx, request.Identity.UserID)
	if errors.Is(err, discordlinks.ErrNotFound) {
		if err = finishIntent(ctx, tx, request.IntentID, request.ResultHash, request.ResultExpiresAt,
			discordlinks.ResultStatusNotLinked, discordlinks.ErrorCodeDiscordUserNotLinked, "", request.Now); err != nil {
			return "", err
		}
		return discordlinks.ResultStatusNotLinked, tx.Commit(ctx)
	}
	if err != nil {
		return "", err
	}
	link, err = scanLink(tx.QueryRow(ctx, `UPDATE discord_links SET username=$2,global_name=$3,
		avatar_hash=$4,scopes=$5,updated_at=$6 WHERE id=$1 RETURNING `+linkColumns,
		link.ID, request.Identity.Username, request.Identity.GlobalName, request.Identity.AvatarHash,
		request.Scopes, request.Now))
	if err != nil {
		return "", err
	}
	if err = finishIntent(ctx, tx, request.IntentID, request.ResultHash, request.ResultExpiresAt,
		discordlinks.ResultStatusAuthenticated, "", link.ID, request.Now); err != nil {
		return "", err
	}
	return discordlinks.ResultStatusAuthenticated, tx.Commit(ctx)
}
