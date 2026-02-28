package artist

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/taibuivan/yomira/internal/platform/database/schema"
	"github.com/taibuivan/yomira/internal/platform/dberr"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (repository *PostgresRepository) ListArtists(context context.Context, f Filter, limit, offset int) ([]*Artist, int, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s IS NULL
	`,
		schema.RefArtist.ID, schema.RefArtist.Name, schema.RefArtist.NameAlt, schema.RefArtist.Bio,
		schema.RefArtist.ImageURL, schema.RefArtist.CreatedAt, schema.RefArtist.UpdatedAt,
		schema.RefArtist.Table, schema.RefArtist.DeletedAt,
	)
	countQuery := fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s IS NULL`, schema.RefArtist.Table, schema.RefArtist.DeletedAt)

	args := []any{}
	countArgs := []any{}

	if f.Query != "" {
		searchTerm := "%" + f.Query + "%"
		query += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		countQuery += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		args = append(args, searchTerm)
		countArgs = append(countArgs, searchTerm)
	}

	query += fmt.Sprintf(" ORDER BY %s ASC LIMIT $", schema.RefArtist.Name) + itos(len(args)+1) + ` OFFSET $` + itos(len(args)+2)
	args = append(args, limit, offset)

	var total int
	if err := repository.db.QueryRow(context, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, dberr.Wrap(err, "count_artists")
	}

	rows, err := repository.db.Query(context, query, args...)
	if err != nil {
		return nil, 0, dberr.Wrap(err, "list_artists")
	}
	defer rows.Close()

	var artists []*Artist
	for rows.Next() {
		a := &Artist{}
		if err := rows.Scan(&a.ID, &a.Name, &a.NameAlt, &a.Bio, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, dberr.Wrap(err, "scan_artist")
		}
		artists = append(artists, a)
	}

	return artists, total, nil
}

func (repository *PostgresRepository) GetArtist(context context.Context, id int) (*Artist, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL
	`,
		schema.RefArtist.ID, schema.RefArtist.Name, schema.RefArtist.NameAlt, schema.RefArtist.Bio,
		schema.RefArtist.ImageURL, schema.RefArtist.CreatedAt, schema.RefArtist.UpdatedAt,
		schema.RefArtist.Table, schema.RefArtist.ID, schema.RefArtist.DeletedAt,
	)

	a := &Artist{}
	err := repository.db.QueryRow(context, query, id).Scan(
		&a.ID, &a.Name, &a.NameAlt, &a.Bio, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt,
	)

	return a, dberr.Wrap(err, "get_artist")
}

func (repository *PostgresRepository) CreateArtist(context context.Context, a *Artist) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING %s, %s, %s
	`,
		schema.RefArtist.Table, schema.RefArtist.Name, schema.RefArtist.NameAlt, schema.RefArtist.Bio,
		schema.RefArtist.ImageURL, schema.RefArtist.CreatedAt, schema.RefArtist.UpdatedAt,
		schema.RefArtist.ID, schema.RefArtist.CreatedAt, schema.RefArtist.UpdatedAt,
	)

	err := repository.db.QueryRow(context, query, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	return dberr.Wrap(err, "create_artist")
}

func (repository *PostgresRepository) UpdateArtist(context context.Context, a *Artist) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = $2, %s = $3, %s = $4, %s = $5, %s = NOW()
		WHERE %s = $1 AND %s IS NULL
		RETURNING %s
	`,
		schema.RefArtist.Table, schema.RefArtist.Name, schema.RefArtist.NameAlt, schema.RefArtist.Bio,
		schema.RefArtist.ImageURL, schema.RefArtist.UpdatedAt, schema.RefArtist.ID, schema.RefArtist.DeletedAt,
		schema.RefArtist.UpdatedAt,
	)

	err := repository.db.QueryRow(context, query, a.ID, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.UpdatedAt)
	return dberr.Wrap(err, "update_artist")
}

func (repository *PostgresRepository) DeleteArtist(context context.Context, id int) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1 AND %s IS NULL`,
		schema.RefArtist.Table, schema.RefArtist.DeletedAt, schema.RefArtist.ID, schema.RefArtist.DeletedAt,
	)

	cmd, err := repository.db.Exec(context, query, id)
	if err != nil {
		return dberr.Wrap(err, "delete_artist")
	}

	if cmd.RowsAffected() == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

func itos(i int) string {
	return strconv.Itoa(i)
}
