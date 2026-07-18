// Package postgres persists verification state in PostgreSQL.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/verification"
)

const groupColumns = `id::text,key,role_id,button_label,button_emoji,button_style,position,enabled,revision,created_at,updated_at`
const membershipColumns = `id::text,user_id,group_id::text,role_id,verified_at,created_at,updated_at`

// Store is the PostgreSQL verification repository.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates a verification repository.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// CreateGroup persists one verification group.
func (store *Store) CreateGroup(ctx context.Context, group verification.Group) (verification.Group, error) {
	return scanGroup(store.pool.QueryRow(ctx, `INSERT INTO verification_groups(key,role_id,button_label,button_emoji,button_style,position,enabled)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+groupColumns,
		group.Key, group.RoleID, group.ButtonLabel, group.ButtonEmoji, group.ButtonStyle, group.Position, group.Enabled))
}

// UpdateGroup replaces one verification group at an expected revision.
func (store *Store) UpdateGroup(ctx context.Context, id string, revision uint64, group verification.Group) (verification.Group, error) {
	updated, err := scanGroup(store.pool.QueryRow(ctx, `UPDATE verification_groups SET key=$3,role_id=$4,button_label=$5,
		button_emoji=$6,button_style=$7,position=$8,enabled=$9,revision=revision+1,updated_at=now()
		WHERE id=$1 AND revision=$2 AND (role_id=$4 OR NOT EXISTS (SELECT 1 FROM verification_memberships WHERE group_id=$1))
		RETURNING `+groupColumns, id, revision, group.Key, group.RoleID,
		group.ButtonLabel, group.ButtonEmoji, group.ButtonStyle, group.Position, group.Enabled))
	if err == verification.ErrNotFound {
		return verification.Group{}, verification.ErrConflict
	}
	return updated, err
}

// GetGroup returns one group by ID.
func (store *Store) GetGroup(ctx context.Context, id string) (verification.Group, error) {
	return scanGroup(store.pool.QueryRow(ctx, `SELECT `+groupColumns+` FROM verification_groups WHERE id=$1`, id))
}

// ListGroups returns all groups in button order.
func (store *Store) ListGroups(ctx context.Context, enabledOnly bool) ([]verification.Group, error) {
	rows, err := store.pool.Query(ctx, `SELECT `+groupColumns+` FROM verification_groups WHERE NOT $1 OR enabled ORDER BY position,id`, enabledOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []verification.Group{}
	for rows.Next() {
		group, scanErr := scanGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// DeleteGroup removes one unused group at an expected revision.
func (store *Store) DeleteGroup(ctx context.Context, id string, revision uint64) error {
	command, err := store.pool.Exec(ctx, `DELETE FROM verification_groups WHERE id=$1 AND revision=$2`, id, revision)
	if err != nil {
		return mapError(err)
	}
	if command.RowsAffected() == 0 {
		return verification.ErrConflict
	}
	return nil
}

// UpsertMembership creates or refreshes one active membership.
func (store *Store) UpsertMembership(ctx context.Context, userID string, group verification.Group) (verification.Membership, error) {
	return scanMembership(store.pool.QueryRow(ctx, `INSERT INTO verification_memberships(user_id,group_id,role_id)
		VALUES($1,$2,$3) ON CONFLICT(user_id,group_id) DO UPDATE SET role_id=excluded.role_id,verified_at=now(),updated_at=now()
		RETURNING `+membershipColumns, userID, group.ID, group.RoleID))
}

// DeleteMembership hard-deletes one active membership.
func (store *Store) DeleteMembership(ctx context.Context, userID, groupID string) (verification.Membership, error) {
	return scanMembership(store.pool.QueryRow(ctx, `DELETE FROM verification_memberships WHERE user_id=$1 AND group_id=$2
		RETURNING `+membershipColumns, userID, groupID))
}

// ListMemberships returns memberships optionally filtered by user.
func (store *Store) ListMemberships(ctx context.Context, userID string) (verification.Page, error) {
	rows, err := store.pool.Query(ctx, `SELECT `+membershipColumns+` FROM verification_memberships WHERE $1='' OR user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return verification.Page{}, err
	}
	defer rows.Close()
	page := verification.Page{Items: []verification.Membership{}}
	for rows.Next() {
		membership, scanErr := scanMembership(rows)
		if scanErr != nil {
			return verification.Page{}, scanErr
		}
		page.Items = append(page.Items, membership)
	}
	page.Total = len(page.Items)
	return page, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanGroup(row scanner) (verification.Group, error) {
	var group verification.Group
	if err := row.Scan(&group.ID, &group.Key, &group.RoleID, &group.ButtonLabel, &group.ButtonEmoji, &group.ButtonStyle, &group.Position, &group.Enabled, &group.Revision, &group.CreatedAt, &group.UpdatedAt); err != nil {
		return verification.Group{}, mapError(err)
	}
	return group, nil
}

func scanMembership(row scanner) (verification.Membership, error) {
	var item verification.Membership
	if err := row.Scan(&item.ID, &item.UserID, &item.GroupID, &item.RoleID, &item.VerifiedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return verification.Membership{}, mapError(err)
	}
	return item, nil
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return verification.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && (postgresError.Code == "23505" || postgresError.Code == "23503") {
		return verification.ErrConflict
	}
	return err
}
