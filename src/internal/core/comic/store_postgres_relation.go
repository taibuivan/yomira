/*
Package comic provides the PostgreSQL implementation for the catalogue's data access.

It utilizes advanced Postgres features to deliver a high-performance discovery experience:
  - Full-Text Search: Uses 'websearch_to_tsquery' and GIN indexes for fuzzy matching.
  - JSON Aggregation: Retrieves complex nested data (e.g., Tags) in a single round-trip.
  - Window Functions: Calculates total result counts without requiring a separate 'COUNT' query.
  - ACID Transactions: Ensures atomicity when updating comics and their junction tables.

The repository follows an "Aggregate" pattern where sub-resources are managed
through the main repository instance to maintain domain integrity.
*/
package comic

import (
	"context"
	"fmt"

	"github.com/taibuivan/yomira/internal/platform/database/schema"
)

// # PostgreSQL Repositories

// # Title & Relation Management

/*
ListTitles returns all alternative titles for a comic.
*/
func (repository *comicRepository) ListTitles(context context.Context, comicID string) ([]*Title, error) {

	// Joint query to resolve language labels
	query := fmt.Sprintf(`
		SELECT t.%s, l.%s, t.%s
		FROM %s t
		JOIN %s l ON t.%s = l.%s
		WHERE t.%s = $1
	`,
		schema.CoreComicTitle.ComicID, schema.RefLanguage.Code, schema.CoreComicTitle.Title,
		schema.CoreComicTitle.Table,
		schema.RefLanguage.Table, schema.CoreComicTitle.LanguageID, schema.RefLanguage.ID,
		schema.CoreComicTitle.ComicID,
	)

	// Execute retrieval
	rows, err := repository.pool.Query(context, query, comicID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list titles failed: %w", err)
	}
	defer rows.Close()

	// Hydrate title slice
	var titles []*Title

	// Iterate rows
	for rows.Next() {
		var title Title
		if err := rows.Scan(&title.ComicID, &title.Language, &title.Title); err != nil {
			return nil, err
		}
		titles = append(titles, &title)
	}

	// Return hydrated title slice
	return titles, nil
}

/*
UpsertTitle creates or updates a language-specific title.
*/
func (repository *comicRepository) UpsertTitle(context context.Context, comicID, langCode, title string) error {

	// Record insertion with language resolution and conflict handling
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s)
		VALUES ($1, (SELECT %s FROM %s WHERE %s = $2), $3)
		ON CONFLICT (%s, %s) DO UPDATE SET %s = EXCLUDED.%s
	`,
		schema.CoreComicTitle.Table, schema.CoreComicTitle.ComicID, schema.CoreComicTitle.LanguageID, schema.CoreComicTitle.Title,
		schema.RefLanguage.ID, schema.RefLanguage.Table, schema.RefLanguage.Code,
		schema.CoreComicTitle.ComicID, schema.CoreComicTitle.LanguageID, schema.CoreComicTitle.Title, schema.CoreComicTitle.Title,
	)

	// Command execution
	_, err := repository.pool.Exec(context, query, comicID, langCode, title)
	if err != nil {
		return fmt.Errorf("postgres: upsert title failed: %w", err)
	}
	return nil
}

/*
DeleteTitle removes a specific language title from a comic.
*/
func (repository *comicRepository) DeleteTitle(context context.Context, comicID, langCode string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE %s = $1 AND %s = (SELECT %s FROM %s WHERE %s = $2)
	`,
		schema.CoreComicTitle.Table,
		schema.CoreComicTitle.ComicID, schema.CoreComicTitle.LanguageID, schema.RefLanguage.ID, schema.RefLanguage.Table, schema.RefLanguage.Code,
	)
	_, err := repository.pool.Exec(context, query, comicID, langCode)
	return err
}

/*
ListRelations fetches all linked publications.
*/
func (repository *comicRepository) ListRelations(context context.Context, comicID string) ([]*Relation, error) {

	// Relation lookup with title resolution
	query := fmt.Sprintf(`
		SELECT r.%s, r.%s, r.%s, c.%s
		FROM %s r
		JOIN %s c ON r.%s = c.%s
		WHERE r.%s = $1
	`,
		schema.CoreComicRelation.FromComicID, schema.CoreComicRelation.ToComicID, schema.CoreComicRelation.RelationType, schema.CoreComic.Title,
		schema.CoreComicRelation.Table,
		schema.CoreComic.Table, schema.CoreComicRelation.ToComicID, schema.CoreComic.ID,
		schema.CoreComicRelation.FromComicID,
	)

	// Query execution
	rows, err := repository.pool.Query(context, query, comicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Hydrate relation slice
	var rels []*Relation
	for rows.Next() {
		var relation Relation
		if err := rows.Scan(&relation.FromComicID, &relation.ToComicID, &relation.Type, &relation.ToTitle); err != nil {
			return nil, err
		}
		rels = append(rels, &relation)
	}
	return rels, nil
}

/*
AddRelation stubs a directional link between two comics.

Returns:
  - error: Database execution errors
*/
func (repository *comicRepository) AddRelation(context context.Context, fromID, toID, relType string) error {

	// Connection insertion logic
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, schema.CoreComicRelation.Table, schema.CoreComicRelation.FromComicID, schema.CoreComicRelation.ToComicID, schema.CoreComicRelation.RelationType)

	// Execution
	_, err := repository.pool.Exec(context, query, fromID, toID, relType)
	return err
}

/*
RemoveRelation deletes a link between two comics.

Returns:
  - error: Database execution errors
*/
func (repository *comicRepository) RemoveRelation(context context.Context, fromID, toID, relType string) error {

	// Delete execution
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1 AND %s = $2 AND %s = $3`,
		schema.CoreComicRelation.Table, schema.CoreComicRelation.FromComicID, schema.CoreComicRelation.ToComicID, schema.CoreComicRelation.RelationType)

	// Command execution
	_, err := repository.pool.Exec(context, query, fromID, toID, relType)
	return err
}
