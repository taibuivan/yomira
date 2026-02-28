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
	"strings"

	"github.com/taibuivan/yomira/internal/platform/database/schema"
	"github.com/taibuivan/yomira/internal/platform/dberr"
)

// # PostgreSQL Repositories

// # Assets (Covers & Art)

/*
ListCovers returns all available cover variants for a comic.

Description: Retrieves metadata for all covers (volumes, seasons, variants).
Results are ordered primarily by volume number, then by creation date.

Parameters:
  - context: context.Context
  - comicID: string (UUID)

Returns:
  - []*Cover: Slice of cover entities
  - error: Storage execution failures
*/
func (repository *comicRepository) ListCovers(context context.Context, comicID string) ([]*Cover, error) {

	// Query Definition
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1
		ORDER BY %s ASC NULLS LAST, %s DESC
	`,
		schema.CoreComicCover.ID, schema.CoreComicCover.ComicID, schema.CoreComicCover.Volume, schema.CoreComicCover.ImageURL, schema.CoreComicCover.Description, schema.CoreComicCover.CreatedAt,
		schema.CoreComicCover.Table,
		schema.CoreComicCover.ComicID,
		schema.CoreComicCover.Volume, schema.CoreComicCover.CreatedAt,
	)

	// Execution
	rows, err := repository.pool.Query(context, query, comicID)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to list covers: %w", err)
	}
	defer rows.Close()

	// Initialisation
	var covers []*Cover

	// Data Mapping Pipeline
	for rows.Next() {
		var cover Cover
		err := rows.Scan(
			&cover.ID,
			&cover.ComicID,
			&cover.Volume,
			&cover.ImageURL,
			&cover.Description,
			&cover.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("postgres: failed to scan cover: %w", err)
		}

		covers = append(covers, &cover)
	}

	return covers, nil
}

/*
AddCover attaches a new volume or variant cover.

Parameters:
  - context: context.Context
  - cover: *Cover (Hydrated entity with ID and ComicID)

Returns:
  - error: Constraint or persistence failures
*/
func (repository *comicRepository) AddCover(context context.Context, cover *Cover) error {

	// Insertion Command
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, $5)
	`, schema.CoreComicCover.Table, schema.CoreComicCover.ID, schema.CoreComicCover.ComicID, schema.CoreComicCover.Volume, schema.CoreComicCover.ImageURL, schema.CoreComicCover.Description)

	// Execution
	_, err := repository.pool.Exec(context, query,
		cover.ID,
		cover.ComicID,
		cover.Volume,
		cover.ImageURL,
		cover.Description,
	)

	if err != nil {
		return fmt.Errorf("postgres: failed to add cover: %w", err)
	}

	return nil
}

/*
DeleteCover removes a specific cover by ID.
*/
func (repository *comicRepository) DeleteCover(context context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1`, schema.CoreComicCover.Table, schema.CoreComicCover.ID)
	_, err := repository.pool.Exec(context, query, id)
	return err
}

/*
ListArt returns the fanart/gallery images for a comic.

Description: Retrieves all uploaded artwork for a comic.
Features a flexible filter for 'approved' status to toggle between
public and moderator views efficiently.

Parameters:
  - context: context.Context
  - comicID: string (UUID)
  - onlyApproved: bool (If true, excludes pending submissions)

Returns:
  - []*Art: Gallery metadata
  - error: Database execution failures
*/
func (repository *comicRepository) ListArt(context context.Context, comicID string, onlyApproved bool) ([]*Art, error) {

	// Dynamic Query Configuration
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1
	`,
		schema.CoreComicArt.ID, schema.CoreComicArt.ComicID, schema.CoreComicArt.UploaderID, schema.CoreComicArt.ImageURL, schema.CoreComicArt.IsApproved, schema.CoreComicArt.CreatedAt,
		schema.CoreComicArt.Table,
		schema.CoreComicArt.ComicID,
	))

	args := []any{comicID}

	// Status Filter Application
	if onlyApproved {
		queryBuilder.WriteString(fmt.Sprintf(" AND %s = true", schema.CoreComicArt.IsApproved))
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s DESC", schema.CoreComicArt.CreatedAt))

	// Pool execution
	rows, err := repository.pool.Query(context, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to list art: %w", err)
	}
	defer rows.Close()

	// Initialisation
	var arts []*Art

	// Hydration Loop
	for rows.Next() {
		var art Art
		err := rows.Scan(
			&art.ID,
			&art.ComicID,
			&art.UploaderID,
			&art.ImageURL,
			&art.IsApproved,
			&art.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("postgres: failed to scan art: %w", err)
		}

		arts = append(arts, &art)
	}

	return arts, nil
}

/*
AddArt persists a new gallery image.

Parameters:
  - context: context.Context
  - art: *Art (Entity with ComicID and UploaderID)

Returns:
  - error: Storage failures
*/
func (repository *comicRepository) AddArt(context context.Context, art *Art) error {

	// Insertion blueprint
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, $5)
	`, schema.CoreComicArt.Table, schema.CoreComicArt.ID, schema.CoreComicArt.ComicID, schema.CoreComicArt.UploaderID, schema.CoreComicArt.ImageURL, schema.CoreComicArt.IsApproved)

	// Contextual Execute
	_, err := repository.pool.Exec(context, query,
		art.ID,
		art.ComicID,
		art.UploaderID,
		art.ImageURL,
		art.IsApproved,
	)

	return dberr.Wrap(err, "add_art")
}

/*
DeleteArt removes a gallery image.
*/
func (repository *comicRepository) DeleteArt(context context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1`, schema.CoreComicArt.Table, schema.CoreComicArt.ID)
	_, err := repository.pool.Exec(context, query, id)
	return err
}

/*
ApproveArt toggles the visibility of a gallery image.

Parameters:
  - context: context.Context
  - id: string (Target Art UUID)
  - approved: bool (Target visibility)

Returns:
  - error: Update failures
*/
func (repository *comicRepository) ApproveArt(context context.Context, id string, approved bool) error {

	// Direct Update definition
	query := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2`, schema.CoreComicArt.Table, schema.CoreComicArt.IsApproved, schema.CoreComicArt.ID)

	// Batch pool execution
	_, err := repository.pool.Exec(context, query, approved, id)

	return dberr.Wrap(err, "approve_art")
}

// EOF
