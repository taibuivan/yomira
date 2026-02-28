package language

import (
	"context"
	"fmt"

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

func (repository *PostgresRepository) ListLanguages(context context.Context) ([]*Language, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s
		FROM %s
		ORDER BY %s ASC;
	`,
		schema.RefLanguage.ID,
		schema.RefLanguage.Code,
		schema.RefLanguage.Name,
		schema.RefLanguage.NativeName,
		schema.RefLanguage.Table,
		schema.RefLanguage.ID,
	)

	rows, err := repository.db.Query(context, query)
	if err != nil {
		return nil, dberr.Wrap(err, "list_languages")
	}
	defer rows.Close()

	var langs []*Language
	for rows.Next() {
		l := &Language{}
		if err := rows.Scan(&l.ID, &l.Code, &l.Name, &l.NativeName); err != nil {
			return nil, dberr.Wrap(err, "scan_language")
		}
		langs = append(langs, l)
	}

	return langs, nil
}

func (repository *PostgresRepository) GetLanguageByCode(context context.Context, code string) (*Language, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s
		FROM %s
		WHERE %s = $1;
	`,
		schema.RefLanguage.ID,
		schema.RefLanguage.Code,
		schema.RefLanguage.Name,
		schema.RefLanguage.NativeName,
		schema.RefLanguage.Table,
		schema.RefLanguage.Code,
	)

	l := &Language{}
	err := repository.db.QueryRow(context, query, code).Scan(&l.ID, &l.Code, &l.Name, &l.NativeName)
	return l, dberr.Wrap(err, "get_language")
}
