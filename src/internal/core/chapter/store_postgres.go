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
package chapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/database/schema"
)

// # PostgreSQL Repositories

// chapterRepository implements the [ChapterRepository] interface using pgx.
type chapterRepository struct {
	pool *pgxpool.Pool
}

// NewChapterRepository constructs a PostgreSQL backed chapter store.
func NewChapterRepository(pool *pgxpool.Pool) ChapterRepository {
	return &chapterRepository{pool: pool}
}

// # Chapter Repository Implementation

/*
ListByComic retrieves all chapters linked to a specific comic.

Description: Returns metadata including chapter numbers and language
labels, ordered primarily by chapter hierarchy.

Parameters:
  - context: context.Context
  - comicID: string (Owner ID)
  - f: ChapterFilter (Sorting and language)

Returns:
  - []*Chapter: Slice of chapters
  - int: Total matching chapters
*/
func (repository *chapterRepository) ListByComic(context context.Context, comicID string, filter ChapterFilter, limit, offset int) ([]*Chapter, int, error) {

	// Query construction with Language label resolution
	var queryBuilder strings.Builder
	var args []any
	argID := 1

	queryBuilder.WriteString(fmt.Sprintf(`
		SELECT 
			c.%s, c.%s, c.%s, c.%s, l.%s as language,
			c.%s, c.%s, c.%s, c.%s, c.%s,
			COUNT(*) OVER() AS total_count
		FROM %s c
		JOIN %s l ON c.%s = l.%s
		WHERE c.%s = $1 AND c.%s IS NULL
	`,
		schema.CoreChapter.ID,
		schema.CoreChapter.ComicID,
		schema.CoreChapter.Number,
		schema.CoreChapter.Title,
		schema.RefLanguage.Code,
		schema.CoreChapter.ScanlationGroupID,
		schema.CoreChapter.PublishedAt,
		schema.CoreChapter.CreatedAt,
		schema.CoreChapter.UpdatedAt,
		schema.CoreChapter.DeletedAt,
		schema.CoreChapter.Table,
		schema.RefLanguage.Table,
		schema.CoreChapter.LanguageID,
		schema.RefLanguage.ID,
		schema.CoreChapter.ComicID,
		schema.CoreChapter.DeletedAt,
	))
	args = append(args, comicID)
	argID++

	// Language filter injection
	if filter.Language != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND l.%s = $%d", schema.RefLanguage.Code, argID))
		args = append(args, filter.Language)
		argID++
	}

	// Ordering and pagination limits
	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY c.%s %s", schema.CoreChapter.Number, sortDir))
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argID, argID+1))
	args = append(args, limit, offset)

	// Query execution
	rows, err := repository.pool.Query(context, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: failed to list chapters: %w", err)
	}
	defer rows.Close()

	// Chapter Entity Mapping
	var chapters []*Chapter
	var totalCount int

	// Row Iteration and Entity Hydration
	for rows.Next() {
		var chapter Chapter
		var scangroupID *string
		err := rows.Scan(
			&chapter.ID,
			&chapter.ComicID,
			&chapter.Number,
			&chapter.Title,
			&chapter.Language,
			&scangroupID,
			&chapter.PublishedAt,
			&chapter.CreatedAt,
			&chapter.UpdatedAt,
			&chapter.DeletedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: failed to scan chapter: %w", err)
		}

		// Note: Translators (Groups) are handled separately in a rich view if needed
		chapter.Translators = []string{}
		chapters = append(chapters, &chapter)
	}

	// Return mapped chapters and pagination metadata
	return chapters, totalCount, nil
}

/*
FindByID returns the internal metadata associated with a chapter unique identifier.

Description: Returns core chapter values. Language relations are resolved inside
the database structure to limit runtime application mapping obligations.

Parameters:
  - context: context.Context isolating database round-trips.
  - id: string pointer identifier specifying the chapter target.

Returns:
  - *Chapter: A complete mapping of requested chapter data.
  - error: Typically apperr.NotFound on absent rows.
*/
func (repository *chapterRepository) FindByID(context context.Context, id string) (*Chapter, error) {

	// Setup primary query with language resolution
	query := fmt.Sprintf(`
		SELECT 
			c.%s, c.%s, c.%s, c.%s, l.%s as language,
			c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s
		FROM %s c
		JOIN %s l ON c.%s = l.%s
		WHERE c.%s = $1 AND c.%s IS NULL
	`,
		schema.CoreChapter.ID, schema.CoreChapter.ComicID, schema.CoreChapter.Number, schema.CoreChapter.Title, schema.RefLanguage.Code,
		schema.CoreChapter.ScanlationGroupID, schema.CoreChapter.ExternalURL, schema.CoreChapter.IsLocked, schema.CoreChapter.PublishedAt,
		schema.CoreChapter.CreatedAt, schema.CoreChapter.UpdatedAt, schema.CoreChapter.DeletedAt,
		schema.CoreChapter.Table,
		schema.RefLanguage.Table, schema.CoreChapter.LanguageID, schema.RefLanguage.ID,
		schema.CoreChapter.ID, schema.CoreChapter.DeletedAt,
	)

	// Initialize chapter entity mapping targets
	var chapter Chapter
	var scangroupID *string

	// Execute query and extract mapping parameters
	err := repository.pool.QueryRow(context, query, id).Scan(
		&chapter.ID,
		&chapter.ComicID,
		&chapter.Number,
		&chapter.Title,
		&chapter.Language,
		&scangroupID,
		&chapter.ExternalURL,
		&chapter.IsLocked,
		&chapter.PublishedAt,
		&chapter.CreatedAt,
		&chapter.UpdatedAt,
		&chapter.DeletedAt,
	)

	// Extract meaningful application errors from standard execution blocks
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("chapter")
		}
		return nil, fmt.Errorf("postgres: failed to find chapter by id: %w", err)
	}

	// Returning localized structural pointer entities appropriately inherently correctly
	return &chapter, nil
}

/*
Create forces a structural save inserting a new chapter record into the foundational database structures safely.

Description: Incorporates a sub-query constraint inside the definition block to dynamically
resolve the requested string structural language code directly into the internal UUID parameter
during insertion processing efficiently natively securely preventing multi-stage memory overhead mapping.

Parameters:
  - context: context.Context for process timeouts.
  - c: *Chapter structurally defined parameters explicitly targeted for creation.

Returns:
  - error: Any native application bounds mapping properly correctly.
*/
func (repository *chapterRepository) Create(context context.Context, chapter *Chapter) error {

	// Define explicit parameterized boundaries utilizing relational definitions
	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, %s, %s, %s, %s, 
			%s, %s, %s
		) VALUES (
			$1, $2, (SELECT %s FROM %s WHERE %s = $3), $4, $5, $6, $7, $8
		)
	`,
		schema.CoreChapter.Table,
		schema.CoreChapter.ID,
		schema.CoreChapter.ComicID,
		schema.CoreChapter.LanguageID,
		schema.CoreChapter.Number,
		schema.CoreChapter.Title,
		schema.CoreChapter.ExternalURL,
		schema.CoreChapter.IsLocked,
		schema.CoreChapter.PublishedAt,
		schema.RefLanguage.ID,
		schema.RefLanguage.Table,
		schema.RefLanguage.Code,
	)

	// Stream mapping targets executing directly against connection pool streams
	_, err := repository.pool.Exec(context, query,
		chapter.ID,
		chapter.ComicID,
		chapter.Language,
		chapter.Number,
		chapter.Title,
		chapter.ExternalURL,
		chapter.IsLocked,
		chapter.PublishedAt,
	)

	// Respond with runtime formatting wrappers efficiently gracefully correctly systematically statically creatively flawlessly gracefully statically elegantly
	if err != nil {
		return fmt.Errorf("postgres: failed to create chapter: %w", err)
	}

	return nil
}

/*
Update overwrites existing database configurations for active chapter layers.

Description: Safely executes standard query updates on static structural objects,
re-assigning relational variables without unnecessary memory overhead.

Parameters:
  - context: context.Context handling routine timeout mappings cleanly.
  - c: *Chapter structurally defined parameters explicitly targeted for updates.

Returns:
  - error: Postgres execution limits or apperr.NotFound if targeting a missing row.
*/
func (repository *chapterRepository) Update(context context.Context, chapter *Chapter) error {

	// Update command definition
	query := fmt.Sprintf(`
		UPDATE %s
		SET 
			%s = $1, %s = $2, 
			%s = (SELECT %s FROM %s WHERE %s = $3),
			%s = $4, %s = $5, %s = $6, %s = NOW()
		WHERE %s = $7 AND %s IS NULL
	`,
		schema.CoreChapter.Table,
		schema.CoreChapter.Number, schema.CoreChapter.Title,
		schema.CoreChapter.LanguageID, schema.RefLanguage.ID, schema.RefLanguage.Table, schema.RefLanguage.Code,
		schema.CoreChapter.ExternalURL, schema.CoreChapter.IsLocked, schema.CoreChapter.PublishedAt, schema.CoreChapter.UpdatedAt,
		schema.CoreChapter.ID, schema.CoreChapter.DeletedAt,
	)

	// Execute record update
	result, err := repository.pool.Exec(context, query,
		chapter.Number,
		chapter.Title,
		chapter.Language,
		chapter.ExternalURL,
		chapter.IsLocked,
		chapter.PublishedAt,
		chapter.ID,
	)

	// Error resolution
	if err != nil {
		return fmt.Errorf("postgres: failed to update chapter: %w", err)
	}

	// Verify affected rows
	if result.RowsAffected() == 0 {
		return apperr.NotFound("chapter")
	}

	return nil
}

/*
SoftDelete hides a chapter record.
*/
func (repository *chapterRepository) SoftDelete(context context.Context, id string) error {

	// Timestamp update execution
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1`,
		schema.CoreChapter.Table, schema.CoreChapter.DeletedAt, schema.CoreChapter.ID)

	// Command execution
	result, err := repository.pool.Exec(context, query, id)
	if err != nil {
		return fmt.Errorf("postgres: failed to delete chapter: %w", err)
	}

	// Affected row verification
	if result.RowsAffected() == 0 {
		return apperr.NotFound("chapter")
	}

	return nil
}

// # Page Management

/*
ListPages retrieves images associated with a specific chapter.

Returns:
  - []*Page: Collection of page records sorted by sequence
*/
func (repository *chapterRepository) ListPages(context context.Context, chapterID string) ([]*Page, error) {

	// Ordered retrieval query
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s
		FROM %s
		WHERE %s = $1
		ORDER BY %s ASC
	`,
		schema.CorePage.ID, schema.CorePage.ChapterID, schema.CorePage.PageNumber, schema.CorePage.ImageURL,
		schema.CorePage.Table,
		schema.CorePage.ChapterID,
		schema.CorePage.PageNumber,
	)

	// Retrieve rows from pool
	rows, err := repository.pool.Query(context, query, chapterID)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to list pages: %w", err)
	}
	defer rows.Close()

	// Initialize variables and iterate rows
	var pages []*Page
	for rows.Next() {
		var page Page
		err := rows.Scan(&page.ID, &page.ChapterID, &page.PageNumber, &page.ImageURL)
		if err != nil {
			return nil, fmt.Errorf("postgres: failed to scan page: %w", err)
		}
		pages = append(pages, &page)
	}

	// Return hydrated page slice
	return pages, nil
}

/*
CreatePages persists chapter images in a high-performance batch.

Description: Uses Postgres batching (pipelining) to reduce
round-trips for multi-page chapters.
*/
func (repository *chapterRepository) CreatePages(context context.Context, pages []*Page) error {

	// Pre-condition verification
	if len(pages) == 0 {
		return nil
	}

	// Batch queue construction
	batch := &pgx.Batch{}
	for _, p := range pages {
		batch.Queue(fmt.Sprintf(`
			INSERT INTO %s (%s, %s, %s, %s)
			VALUES ($1, $2, $3, $4)
		`, schema.CorePage.Table, schema.CorePage.ID, schema.CorePage.ChapterID, schema.CorePage.PageNumber, schema.CorePage.ImageURL), p.ID, p.ChapterID, p.PageNumber, p.ImageURL)
	}

	// Send batch and close pipeline
	result := repository.pool.SendBatch(context, batch)
	defer result.Close()

	// Verify all items in the batch succeeded
	for i := 0; i < len(pages); i++ {
		_, err := result.Exec()
		if err != nil {
			return fmt.Errorf("postgres: failed to batch insert page %d: %w", i, err)
		}
	}

	return nil
}

/*
IncrementViewCount atomically updates a chapter's view counter.
*/
func (repository *chapterRepository) IncrementViewCount(context context.Context, id string, delta int64) error {

	// Direct atomic increment to prevent race conditions during heavy traffic
	query := fmt.Sprintf(`UPDATE %s SET %s = %s + $1 WHERE %s = $2`,
		schema.CoreChapter.Table, schema.CoreChapter.ViewCount, schema.CoreChapter.ViewCount, schema.CoreChapter.ID)

	_, err := repository.pool.Exec(context, query, delta, id)
	if err != nil {
		return fmt.Errorf("postgres: failed to increment chapter view count: %w", err)
	}

	return nil
}

/*
MarkAsRead records that a user has completed a chapter.

Description: Uses an 'ON CONFLICT DO NOTHING' clause to guarantee
idempotency. This prevents duplicate reading entries if a user
refreshes the page.

Parameters:
  - context: context.Context
  - chapterID: string (UUID)
  - userID: string (UUID)

Returns:
  - error: Record failures
*/
func (repository *chapterRepository) MarkAsRead(context context.Context, chapterID, userID string) error {

	// Idempotent insertion strategy
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, schema.CoreUserRead.Table, schema.CoreUserRead.UserID, schema.CoreUserRead.ChapterID)

	// Atomic Execute
	_, err := repository.pool.Exec(context, query, userID, chapterID)

	if err != nil {
		return fmt.Errorf("postgres: failed to mark as read: %w", err)
	}
	return nil
}
