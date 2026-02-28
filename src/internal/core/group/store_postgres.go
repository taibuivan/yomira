// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package group

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/taibuivan/yomira/internal/platform/database/schema"
	"github.com/taibuivan/yomira/internal/platform/dberr"
)

// PostgresRepository implements [Repository] using pgx.
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository constructs a PostgreSQL backed group store.
func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// # Group Retrieval

/*
List returns a filtered and paginated list of groups.

Description: Uses trigram ILIKE for entity search and COUNT(*) OVER() for total metadata.

Parameters:
  - context: context.Context
  - filter: Filter
  - limit: int
  - offset: int

Returns:
  - []*Group: Slice of matching groups
  - int: Total record count
  - error: Database retrieval failures
*/
func (repository *PostgresRepository) List(context context.Context, filter Filter, limit, offset int) ([]*Group, int, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf(`
		SELECT 
			%s, %s, %s, %s, %s, %s,
			%s, %s, %s, %s,
			COUNT(*) OVER() as total
		FROM %s
		WHERE %s IS NULL
	`,
		schema.CoreGroup.ID,
		schema.CoreGroup.Name,
		schema.CoreGroup.Slug,
		schema.CoreGroup.Description,
		schema.CoreGroup.Website,
		schema.CoreGroup.IsOfficialPublisher,
		schema.CoreGroup.IsActive,
		schema.CoreGroup.FollowCount,
		schema.CoreGroup.CreatedAt,
		schema.CoreGroup.UpdatedAt,
		schema.CoreGroup.Table,
		schema.CoreGroup.DeletedAt,
	))

	args := []any{}
	argID := 1

	if filter.Query != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND %s ILIKE $%d", schema.CoreGroup.Name, argID))
		args = append(args, "%"+filter.Query+"%")
		argID++
	}

	if filter.IsOfficialPublisher != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND %s = $%d", schema.CoreGroup.IsOfficialPublisher, argID))
		args = append(args, *filter.IsOfficialPublisher)
		argID++
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s ASC LIMIT $%d OFFSET $%d", schema.CoreGroup.Name, argID, argID+1))
	args = append(args, limit, offset)

	rows, err := repository.db.Query(context, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, dberr.Wrap(err, "list_groups")
	}
	defer rows.Close()

	var groups []*Group
	var total int
	for rows.Next() {
		group := &Group{}
		err := rows.Scan(
			&group.ID, &group.Name, &group.Slug, &group.Description, &group.Website, &group.IsOfficialPublisher,
			&group.IsActive, &group.FollowCount, &group.CreatedAt, &group.UpdatedAt, &total,
		)
		if err != nil {
			return nil, 0, dberr.Wrap(err, "scan_group")
		}
		groups = append(groups, group)
	}

	return groups, total, nil
}

/*
FindByID retrieves a single group record by its primary key.

Parameters:
  - context: context.Context
  - id: string

Returns:
  - *Group: Hydrated entity
  - error: Database retrieval failures
*/
func (repository *PostgresRepository) FindByID(context context.Context, id string) (*Group, error) {
	query := fmt.Sprintf(`
		SELECT 
			%s, %s, %s, %s, %s, %s, %s,
			%s, %s, %s, %s, %s,
			%s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL
	`,
		schema.CoreGroup.ID,
		schema.CoreGroup.Name,
		schema.CoreGroup.Slug,
		schema.CoreGroup.Description,
		schema.CoreGroup.Website,
		schema.CoreGroup.Discord,
		schema.CoreGroup.Twitter,
		schema.CoreGroup.Patreon,
		schema.CoreGroup.Youtube,
		schema.CoreGroup.MangaUpdates,
		schema.CoreGroup.IsOfficialPublisher,
		schema.CoreGroup.IsActive,
		schema.CoreGroup.IsFocused,
		schema.CoreGroup.VerifiedAt,
		schema.CoreGroup.CreatedAt,
		schema.CoreGroup.UpdatedAt,
		schema.CoreGroup.Table,
		schema.CoreGroup.ID, schema.CoreGroup.DeletedAt,
	)
	group := &Group{}
	err := repository.db.QueryRow(context, query, id).Scan(
		&group.ID, &group.Name, &group.Slug, &group.Description, &group.Website, &group.Discord, &group.Twitter,
		&group.Patreon, &group.Youtube, &group.MangaUpdates, &group.IsOfficialPublisher, &group.IsActive,
		&group.IsFocused, &group.VerifiedAt, &group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return nil, dberr.Wrap(err, "get_group_by_id")
	}
	return group, nil
}

/*
FindBySlug retrieves a group by its unique URL slug.

Parameters:
  - context: context.Context
  - slug: string

Returns:
  - *Group: Hydrated entity
  - error: Database retrieval failures
*/
func (repository *PostgresRepository) FindBySlug(context context.Context, slug string) (*Group, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL`,
		schema.CoreGroup.ID,
		schema.CoreGroup.Name,
		schema.CoreGroup.Slug,
		schema.CoreGroup.Description,
		schema.CoreGroup.IsActive,
		schema.CoreGroup.CreatedAt,
		schema.CoreGroup.Table,
		schema.CoreGroup.Slug,
		schema.CoreGroup.DeletedAt,
	)
	group := &Group{}
	err := repository.db.QueryRow(context, query, slug).Scan(
		&group.ID, &group.Name, &group.Slug, &group.Description, &group.IsActive, &group.CreatedAt,
	)
	if err != nil {
		return nil, dberr.Wrap(err, "get_group_by_slug")
	}
	return group, nil
}

// # Group Mutation

/*
Create inserts a new group record.

Parameters:
  - context: context.Context
  - group: *Group

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) Create(context context.Context, group *Group) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, %s, %s, %s, %s, %s, %s, %s
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING %s, %s`,
		schema.CoreGroup.Table,
		schema.CoreGroup.ID,
		schema.CoreGroup.Name,
		schema.CoreGroup.Slug,
		schema.CoreGroup.Description,
		schema.CoreGroup.Website,
		schema.CoreGroup.IsActive,
		schema.CoreGroup.CreatedAt,
		schema.CoreGroup.UpdatedAt,
		schema.CoreGroup.CreatedAt,
		schema.CoreGroup.UpdatedAt,
	)
	err := repository.db.QueryRow(context, query,
		group.ID, group.Name, group.Slug, group.Description, group.Website, group.IsActive,
	).Scan(&group.CreatedAt, &group.UpdatedAt)

	return dberr.Wrap(err, "create_group")
}

/*
Update modifies group metadata fields.

Parameters:
  - context: context.Context
  - group: *Group

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) Update(context context.Context, group *Group) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = $2, %s = $3, %s = NOW()
		WHERE %s = $1 AND %s IS NULL
		RETURNING %s`,
		schema.CoreGroup.Table,
		schema.CoreGroup.Description,
		schema.CoreGroup.Website,
		schema.CoreGroup.UpdatedAt,
		schema.CoreGroup.ID,
		schema.CoreGroup.DeletedAt,
		schema.CoreGroup.UpdatedAt,
	)
	err := repository.db.QueryRow(context, query, group.ID, group.Description, group.Website).Scan(&group.UpdatedAt)
	return dberr.Wrap(err, "update_group")
}

/*
SoftDelete flags a group as deleted.

Parameters:
  - context: context.Context
  - id: string

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) SoftDelete(context context.Context, id string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1`,
		schema.CoreGroup.Table, schema.CoreGroup.DeletedAt, schema.CoreGroup.ID)
	_, err := repository.db.Exec(context, query, id)
	return dberr.Wrap(err, "delete_group")
}

// # Membership Implementation

/*
ListMembers retrieves all affiliated users and their roles.

Parameters:
  - context: context.Context
  - groupID: string

Returns:
  - []*Member: List of affiliated users
  - error: Retrieval failures
*/
func (repository *PostgresRepository) ListMembers(context context.Context, groupID string) ([]*Member, error) {
	query := fmt.Sprintf(`
		SELECT m.%s, m.%s, u.%s, m.%s, m.%s
		FROM %s m
		JOIN %s u ON m.%s = u.%s
		WHERE m.%s = $1
		ORDER BY m.%s ASC
	`,
		schema.CoreMember.GroupID, schema.CoreMember.UserID, schema.UserAccount.Username, schema.CoreMember.Role, schema.CoreMember.JoinedAt,
		schema.CoreMember.Table,
		schema.UserAccount.Table, schema.CoreMember.UserID, schema.UserAccount.ID,
		schema.CoreMember.GroupID,
		schema.CoreMember.JoinedAt,
	)
	rows, err := repository.db.Query(context, query, groupID)
	if err != nil {
		return nil, dberr.Wrap(err, "list_group_members")
	}
	defer rows.Close()

	var members []*Member
	for rows.Next() {
		member := &Member{}
		if err := rows.Scan(&member.GroupID,
			&member.UserID,
			&member.Username,
			&member.Role,
			&member.JoinedAt); err != nil {
			return nil, dberr.Wrap(err, "scan_member")
		}
		members = append(members, member)
	}

	return members, nil
}

/*
AddMember inserts a new membership record.

Parameters:
  - context: context.Context
  - member: *Member

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) AddMember(context context.Context, member *Member) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s)
		VALUES ($1, $2, $3, NOW())
		RETURNING %s
	`,
		schema.CoreMember.Table,
		schema.CoreMember.GroupID, schema.CoreMember.UserID, schema.CoreMember.Role, schema.CoreMember.JoinedAt,
		schema.CoreMember.JoinedAt,
	)
	err := repository.db.QueryRow(
		context,
		query,
		member.GroupID,
		member.UserID,
		member.Role,
	).Scan(&member.JoinedAt)
	return dberr.Wrap(err, "add_group_member")
}

/*
UpdateMemberRole modifies a user's role.

Parameters:
  - context: context.Context
  - groupID: string
  - userID: string
  - role: Role

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) UpdateMemberRole(context context.Context, groupID, userID string, role Role) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = $3 WHERE %s = $1 AND %s = $2`,
		schema.CoreMember.Table, schema.CoreMember.Role, schema.CoreMember.GroupID, schema.CoreMember.UserID)
	_, err := repository.db.Exec(context, query, groupID, userID, role)
	return dberr.Wrap(err, "update_member_role")
}

/*
RemoveMember hard-deletes a membership link.

Parameters:
  - context: context.Context
  - groupID: string
  - userID: string

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) RemoveMember(context context.Context, groupID, userID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1 AND %s = $2`,
		schema.CoreMember.Table, schema.CoreMember.GroupID, schema.CoreMember.UserID)
	_, err := repository.db.Exec(context, query, groupID, userID)
	return dberr.Wrap(err, "remove_member")
}

// # Social & Following Implementation

/*
Follow establishes a secure link between a user and a scanlation group.

Description: Executes within an ACID transaction to guarantee atomicity.
1. Inserts a new row into core.scanlationgroupfollow (Idempotent).
2. Atomically increments the group's global followcount.
Roll back completely if any stage fails to prevent counter drift.

Parameters:
  - context: context.Context handling process isolation
  - groupID: string Target UUID
  - userID: string Actor UUID

Returns:
  - error: Transactional or database failures
*/
func (repository *PostgresRepository) Follow(context context.Context, groupID, userID string) error {

	// Establish Transactional Boundary
	transaction, err := repository.db.Begin(context)
	if err != nil {
		return dberr.Wrap(err, "begin_follow_tx")
	}
	defer transaction.Rollback(context)

	// Step 1: Persist Follow Relation
	// Uses ON CONFLICT DO NOTHING to ensure idempotency
	followQuery := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s)
		VALUES ($1, $2, NOW())
		ON CONFLICT DO NOTHING
	`, schema.CoreFollow.Table, schema.CoreFollow.GroupID, schema.CoreFollow.UserID, schema.CoreFollow.CreatedAt)
	_, err = transaction.Exec(context, followQuery, groupID, userID)
	if err != nil {
		return dberr.Wrap(err, "insert_follow")
	}

	// Step 2: Atomic Metric Jump
	countQuery := fmt.Sprintf(`
		UPDATE %s 
		SET %s = %s + 1 
		WHERE %s = $1
	`, schema.CoreGroup.Table, schema.CoreGroup.FollowCount, schema.CoreGroup.FollowCount, schema.CoreGroup.ID)
	_, err = transaction.Exec(context, countQuery, groupID)
	if err != nil {
		return dberr.Wrap(err, "increment_group_follow")
	}

	// Persist Atomic Changeset
	return transaction.Commit(context)
}

/*
Unfollow removes a user-group link and decrements metrics accurately.

Description: Wraps removal and counter decrement in a transaction.
Only decrements if a record was actually removed to prevent negative drift
during concurrent or duplicate requests.

Parameters:
  - context: context.Context
  - groupID: string
  - userID: string

Returns:
  - error: Database or transactional errors
*/
func (repository *PostgresRepository) Unfollow(context context.Context, groupID, userID string) error {

	// Transactional State Setup
	transaction, err := repository.db.Begin(context)
	if err != nil {
		return dberr.Wrap(err, "begin_unfollow_tx")
	}
	defer transaction.Rollback(context)

	// Step 1: Remove Relationship
	delQuery := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE %s = $1 AND %s = $2
	`, schema.CoreFollow.Table, schema.CoreFollow.GroupID, schema.CoreFollow.UserID)
	result, err := transaction.Exec(context, delQuery, groupID, userID)
	if err != nil {
		return dberr.Wrap(err, "delete_follow")
	}

	// Step 2: Validated Counter Decrement
	// Prevents counter from dropping below zero using GREATEST(0, x)
	if result.RowsAffected() > 0 {
		decQuery := fmt.Sprintf(`
			UPDATE %s 
			SET %s = GREATEST(0, %s - 1) 
			WHERE %s = $1
		`, schema.CoreGroup.Table, schema.CoreGroup.FollowCount, schema.CoreGroup.FollowCount, schema.CoreGroup.ID)
		_, err = transaction.Exec(context, decQuery, groupID)
		if err != nil {
			return dberr.Wrap(err, "decrement_group_follow")
		}
	}

	return transaction.Commit(context)
}
