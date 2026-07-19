package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/discordlinks"
)

const intentColumns = `id, kind, COALESCE(subject,''), completion_key, status, expires_at, created_at`
const linkColumns = `id, subject, discord_user_id, username, global_name, avatar_hash, scopes,
	linked_at, unlinked_at, created_at, updated_at`

// Store is the PostgreSQL Discord account-link repository.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates a PostgreSQL Discord account-link repository.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type scanner interface{ Scan(...any) error }

func scanIntent(row scanner) (discordlinks.Intent, error) {
	var intent discordlinks.Intent
	if err := row.Scan(&intent.ID, &intent.Kind, &intent.Subject, &intent.CompletionKey, &intent.Status,
		&intent.ExpiresAt, &intent.CreatedAt); err != nil {
		return discordlinks.Intent{}, mapNotFound(err)
	}
	return intent, nil
}

func scanLink(row scanner) (discordlinks.Link, error) {
	var link discordlinks.Link
	if err := row.Scan(&link.ID, &link.Subject, &link.DiscordUserID, &link.Username,
		&link.GlobalName, &link.AvatarHash, &link.Scopes, &link.LinkedAt,
		&link.UnlinkedAt, &link.CreatedAt, &link.UpdatedAt); err != nil {
		return discordlinks.Link{}, mapNotFound(err)
	}
	link.Status = discordlinks.LinkStatusLinked
	if link.UnlinkedAt != nil {
		link.Status = discordlinks.LinkStatusUnlinked
	}
	if link.AvatarHash != "" {
		extension := "png"
		if strings.HasPrefix(link.AvatarHash, "a_") {
			extension = "gif"
		}
		link.AvatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s?size=128",
			link.DiscordUserID, link.AvatarHash, extension)
	}
	return link, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return discordlinks.ErrNotFound
	}
	return err
}
