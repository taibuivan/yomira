package author

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

func (repository *PostgresRepository) ListAuthors(context context.Context, f Filter, limit, offset int) ([]*Author, int, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s IS NULL
	`,
		schema.RefAuthor.ID, schema.RefAuthor.Name, schema.RefAuthor.NameAlt, schema.RefAuthor.Bio,
		schema.RefAuthor.ImageURL, schema.RefAuthor.CreatedAt, schema.RefAuthor.UpdatedAt,
		schema.RefAuthor.Table, schema.RefAuthor.DeletedAt,
	)
	countQuery := fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s IS NULL`, schema.RefAuthor.Table, schema.RefAuthor.DeletedAt)

	args := []any{}
	countArgs := []any{}

	if f.Query != "" {
		searchTerm := "%" + f.Query + "%"
		query += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		countQuery += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		args = append(args, searchTerm)
		countArgs = append(countArgs, searchTerm)
	}

	query += fmt.Sprintf(" ORDER BY %s ASC LIMIT $", schema.RefAuthor.Name) + itos(len(args)+1) + ` OFFSET $` + itos(len(args)+2)
	args = append(args, limit, offset)

	var total int
	if err := repository.db.QueryRow(context, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, dberr.Wrap(err, "count_authors")
	}

	rows, err := repository.db.Query(context, query, args...)
	if err != nil {
		return nil, 0, dberr.Wrap(err, "list_authors")
	}
	defer rows.Close()

	var authors []*Author
	for rows.Next() {
		a := &Author{}
		if err := rows.Scan(&a.ID, &a.Name, &a.NameAlt, &a.Bio, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, dberr.Wrap(err, "scan_author")
		}
		authors = append(authors, a)
	}

	return authors, total, nil
}

func (repository *PostgresRepository) GetAuthor(context context.Context, id int) (*Author, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL
	`,
		schema.RefAuthor.ID, schema.RefAuthor.Name, schema.RefAuthor.NameAlt, schema.RefAuthor.Bio,
		schema.RefAuthor.ImageURL, schema.RefAuthor.CreatedAt, schema.RefAuthor.UpdatedAt,
		schema.RefAuthor.Table, schema.RefAuthor.ID, schema.RefAuthor.DeletedAt,
	)
	a := &Author{}

	err := repository.db.QueryRow(context, query, id).Scan(
		&a.ID, &a.Name, &a.NameAlt, &a.Bio, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt,
	)

	return a, dberr.Wrap(err, "get_author")
}

func (repository *PostgresRepository) CreateAuthor(context context.Context, a *Author) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING %s, %s, %s
	`,
		schema.RefAuthor.Table, schema.RefAuthor.Name, schema.RefAuthor.NameAlt, schema.RefAuthor.Bio,
		schema.RefAuthor.ImageURL, schema.RefAuthor.CreatedAt, schema.RefAuthor.UpdatedAt,
		schema.RefAuthor.ID, schema.RefAuthor.CreatedAt, schema.RefAuthor.UpdatedAt,
	)

	err := repository.db.QueryRow(context, query, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	return dberr.Wrap(err, "create_author")
}

func (repository *PostgresRepository) UpdateAuthor(context context.Context, a *Author) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = $2, %s = $3, %s = $4, %s = $5, %s = NOW()
		WHERE %s = $1 AND %s IS NULL
		RETURNING %s
	`,
		schema.RefAuthor.Table, schema.RefAuthor.Name, schema.RefAuthor.NameAlt, schema.RefAuthor.Bio,
		schema.RefAuthor.ImageURL, schema.RefAuthor.UpdatedAt, schema.RefAuthor.ID, schema.RefAuthor.DeletedAt,
		schema.RefAuthor.UpdatedAt,
	)

	err := repository.db.QueryRow(context, query, a.ID, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.UpdatedAt)
	return dberr.Wrap(err, "update_author")
}

func (repository *PostgresRepository) DeleteAuthor(context context.Context, id int) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1 AND %s IS NULL`,
		schema.RefAuthor.Table, schema.RefAuthor.DeletedAt, schema.RefAuthor.ID, schema.RefAuthor.DeletedAt,
	)

	cmd, err := repository.db.Exec(context, query, id)
	if err != nil {
		return dberr.Wrap(err, "delete_author")
	}

	if cmd.RowsAffected() == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

func itos(i int) string {
	return strconv.Itoa(i)
}
