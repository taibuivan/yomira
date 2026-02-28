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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/database/schema"
)

// # PostgreSQL Repositories

// comicRepository implements the [ComicRepository] interface using pgx.
type comicRepository struct {
	pool *pgxpool.Pool
}

// NewComicRepository constructs a PostgreSQL backed comic store.
func NewComicRepository(pool *pgxpool.Pool) ComicRepository {
	return &comicRepository{pool: pool}
}

// # Comic Repository Implementation

/*
List returns a filtered, paginated slice of comics and the total count.

Description: This high-performance query utilizes several PostgreSQL advanced
features:
  - Window Function: Uses COUNT(*) OVER() to retrieve total record counts
    without a second query.
  - JSON Aggregation: Sub-queries aggregate associated Tags into a JSON
    array to prevent N+1 overhead.
  - Set Operations: Uses ANY($n) and array operators (&&, <@) for complex
    tag and author filtering.

Parameters:
  - context: context.Context
  - f: ComicFilter (Search, status, tags, sorting)
  - limit: int
  - offset: int

Returns:
  - []*Comic: Slice of hydrated comic entities
  - int: Total count matching filters
  - error: Database execution errors
*/
func (repository *comicRepository) List(context context.Context, filter Filter, limit, offset int) ([]*Comic, int, error) {

	// Query build initialization
	var queryBuilder strings.Builder
	var args []any
	argID := 1

	// Using Window Function to get total count
	// We also aggregate tags into a JSON array for efficient retrieval
	queryBuilder.WriteString(fmt.Sprintf(`
		SELECT 
			c.%s, c.%s, c.%s, c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, c.%s,
			COUNT(*) OVER() AS total_count,
			COALESCE((
				SELECT json_agg(json_build_object('id', t.%s, 'name', t.%s, 'slug', t.%s))
				FROM %s t
				JOIN %s ct ON t.%s = ct.%s
				WHERE ct.%s = c.%s
			), '[]') as tags
		FROM %s c
		WHERE c.%s IS NULL
	`,
		schema.CoreComic.ID,
		schema.CoreComic.Title,
		schema.CoreComic.AltTitle,
		schema.CoreComic.Slug,
		schema.CoreComic.Description,
		schema.CoreComic.CoverURL,
		schema.CoreComic.Status,
		schema.CoreComic.ContentRating,
		schema.CoreComic.Demographic,
		schema.CoreComic.DefaultReadMode,
		schema.CoreComic.OriginLanguage,
		schema.CoreComic.Year,
		schema.CoreComic.ViewCount,
		schema.CoreComic.FollowCount,
		schema.CoreComic.RatingAvg,
		schema.CoreComic.RatingBayesian,
		schema.CoreComic.RatingCount,
		schema.CoreComic.IsLocked,
		schema.CoreComic.CreatedAt,
		schema.CoreComic.UpdatedAt,
		schema.CoreComic.DeletedAt,
		schema.RefTag.ID,
		schema.RefTag.Name,
		schema.RefTag.Slug,
		schema.RefTag.Table,
		schema.ComicTag.Table,
		schema.RefTag.ID,
		schema.ComicTag.TagID,
		schema.ComicTag.ComicID, schema.CoreComic.ID,
		schema.CoreComic.Table,
		schema.CoreComic.DeletedAt,
	))

	// Apply Filters (Dynamic WHERE clause construction)
	if len(filter.Status) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.%s = ANY($%d)", schema.CoreComic.Status, argID))
		args = append(args, filter.Status)
		argID++
	}

	// Content Rating Filtering
	if len(filter.ContentRating) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.%s = ANY($%d)", schema.CoreComic.ContentRating, argID))
		args = append(args, filter.ContentRating)
		argID++
	}

	// Demographic Filtering
	if len(filter.Demographic) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.%s = ANY($%d)", schema.CoreComic.Demographic, argID))
		args = append(args, filter.Demographic)
		argID++
	}

	// Origin Language Filtering
	if len(filter.OriginLanguage) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.%s = ANY($%d)", schema.CoreComic.OriginLanguage, argID))
		args = append(args, filter.OriginLanguage)
		argID++
	}

	// Year Filtering
	if filter.Year != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.%s = $%d", schema.CoreComic.Year, argID))
		args = append(args, *filter.Year)
		argID++
	}

	// Search Query Filtering
	if filter.Query != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.%s @@ websearch_to_tsquery('simple', unaccent($%d))", schema.CoreComic.SearchVector, argID))
		args = append(args, filter.Query)
		argID++
	}

	// Tag Filtering (AND logic for included, NOT EXISTS for excluded)
	if len(filter.IncludedTags) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND $%d::int[] <@ (SELECT array_agg(%s) FROM %s WHERE %s = c.%s)`, argID, schema.ComicTag.TagID, schema.ComicTag.Table, schema.ComicTag.ComicID, schema.CoreComic.ID))
		args = append(args, filter.IncludedTags)
		argID++
	}

	// Excluded Tags Filtering
	if len(filter.ExcludedTags) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND NOT ($%d::int[] && (SELECT array_agg(%s) FROM %s WHERE %s = c.%s))`, argID, schema.ComicTag.TagID, schema.ComicTag.Table, schema.ComicTag.ComicID, schema.CoreComic.ID))
		args = append(args, filter.ExcludedTags)
		argID++
	}

	// Author/Artist Filtering
	// Included Authors Filtering
	if len(filter.IncludedAuthors) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND $%d::int[] && (SELECT array_agg(%s) FROM %s WHERE %s = c.%s)`, argID, schema.ComicAuthor.AuthorID, schema.ComicAuthor.Table, schema.ComicAuthor.ComicID, schema.CoreComic.ID))
		args = append(args, filter.IncludedAuthors)
		argID++
	}

	// Included Artists Filtering
	if len(filter.IncludedArtists) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND $%d::int[] && (SELECT array_agg(%s) FROM %s WHERE %s = c.%s)`, argID, schema.ComicArtist.ArtistID, schema.ComicArtist.Table, schema.ComicArtist.ComicID, schema.CoreComic.ID))
		args = append(args, filter.IncludedArtists)
		argID++
	}

	// Apply Sorting Logic
	sort := fmt.Sprintf("c.%s", schema.CoreComic.CreatedAt) // default
	switch filter.Sort {
	// Popularity
	case "popular":
		sort = fmt.Sprintf("c.%s", schema.CoreComic.ViewCount)
	// Rating
	case "rating":
		sort = fmt.Sprintf("c.%s", schema.CoreComic.RatingBayesian)
	// Follow Count
	case "followcount":
		sort = fmt.Sprintf("c.%s", schema.CoreComic.FollowCount)
	// Alphabetical Order
	case "az":
		sort = fmt.Sprintf("c.%s", schema.CoreComic.Title)
	case "za":
		sort = fmt.Sprintf("c.%s", schema.CoreComic.Title)
	// Latest
	case "latest":
		sort = fmt.Sprintf("c.%s", schema.CoreComic.CreatedAt)
	}

	// Apply Sorting Direction
	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" || filter.Sort == "az" {
		sortDir = "ASC"
	}
	if filter.Sort == "za" {
		sortDir = "DESC"
	}

	// Apply Sorting
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s, c.%s DESC", sort, sortDir, schema.CoreComic.ID))

	// Pagination injection
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argID, argID+1))
	args = append(args, limit, offset)

	// Query Execution
	rows, err := repository.pool.Query(context, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: failed to list comics: %w", err)
	}
	defer rows.Close()

	// Initialize variables
	var comics []*Comic
	var totalCount int

	// Iterate over rows
	for rows.Next() {
		comic := &Comic{}
		var tagsJSON []byte
		err := rows.Scan(
			&comic.ID,
			&comic.Title,
			&comic.TitleAlt,
			&comic.Slug,
			&comic.Synopsis,
			&comic.CoverURL,
			&comic.Status,
			&comic.ContentRating,
			&comic.Demographic,
			&comic.DefaultReadMode,
			&comic.OriginLanguage,
			&comic.Year,
			&comic.ViewCount,
			&comic.FollowCount,
			&comic.RatingAvg,
			&comic.RatingBayesian,
			&comic.RatingCount,
			&comic.IsLocked,
			&comic.CreatedAt,
			&comic.UpdatedAt,
			&comic.DeletedAt,
			&totalCount,
			&tagsJSON,
		)

		// Check for errors during row scanning
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: failed to scan comic: %w", err)
		}

		// Unmarshal tags JSON
		if err := json.Unmarshal(tagsJSON, &comic.Tags); err != nil {
			return nil, 0, fmt.Errorf("postgres: failed to unmarshal tags: %w", err)
		}

		comics = append(comics, comic)
	}

	// Return the list of comics and total count
	return comics, totalCount, nil
}

/*
FindByID retrieves a comic record by its primary key.

Description: Performs a single-row lookup to retrieve core comic metadata.
In addition to the core fields, this query utilizes PostgreSQL's JSON
aggregation capabilities (json_agg and json_build_object) natively to
retrieve the associated tags in a single database round-trip. This avoids
the classic N+1 query problem and optimizes domain hydration performance
for standard application workflows.

Parameters:
  - context: context.Context for request scoping, deadlines, and cancellation tracking
  - id: string representing the UUID primary key of the target comic

Returns:
  - *Comic: The fully hydrated comic entity (including tags), or nil if not found
  - error: Returns apperr.NotFound if the comic does not exist, or an internal error upon failure
*/
func (repository *comicRepository) FindByID(context context.Context, id string) (*Comic, error) {

	// Unified Lookup Query with JSON Tag Aggregation
	// Employs a sub-query utilizing json_agg to merge normalized tagged links
	// directly into the root row response, vastly improving fetch speeds.
	query := fmt.Sprintf(`
		SELECT 
			c.%s, c.%s, c.%s, c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, c.%s,
			c.%s,
			COALESCE((
				SELECT json_agg(json_build_object('id', t.%s, 'name', t.%s, 'slug', t.%s))
				FROM %s t
				JOIN %s ct ON t.%s = ct.%s
				WHERE ct.%s = c.%s
			), '[]') as tags
		FROM %s c
		WHERE c.%s = $1 AND c.%s IS NULL
	`,
		schema.CoreComic.ID,
		schema.CoreComic.Title,
		schema.CoreComic.AltTitle,
		schema.CoreComic.Slug,
		schema.CoreComic.Description,
		schema.CoreComic.CoverURL,
		schema.CoreComic.Status,
		schema.CoreComic.ContentRating,
		schema.CoreComic.Demographic,
		schema.CoreComic.DefaultReadMode,
		schema.CoreComic.OriginLanguage,
		schema.CoreComic.Year,
		schema.CoreComic.ViewCount,
		schema.CoreComic.FollowCount,
		schema.CoreComic.RatingAvg,
		schema.CoreComic.RatingBayesian,
		schema.CoreComic.RatingCount,
		schema.CoreComic.IsLocked,
		schema.CoreComic.CreatedAt,
		schema.CoreComic.UpdatedAt,
		schema.CoreComic.DeletedAt,
		schema.CoreComic.Links,
		schema.RefTag.ID,
		schema.RefTag.Name,
		schema.RefTag.Slug,
		schema.RefTag.Table, schema.ComicTag.Table, schema.RefTag.ID, schema.ComicTag.TagID,
		schema.ComicTag.ComicID, schema.CoreComic.ID,
		schema.CoreComic.Table,
		schema.CoreComic.ID, schema.CoreComic.DeletedAt,
	)

	// Variable Initialization and Record Scanning Pipeline
	// Allocates the local comic instance and a raw byte slice for dynamic JSON processing
	comic := &Comic{}
	var tagsJSON []byte

	// Execute the structured query on the connection pool and map columns dynamically
	err := repository.pool.QueryRow(context, query, id).Scan(
		&comic.ID,
		&comic.Title,
		&comic.TitleAlt,
		&comic.Slug,
		&comic.Synopsis,
		&comic.CoverURL,
		&comic.Status,
		&comic.ContentRating,
		&comic.Demographic,
		&comic.DefaultReadMode,
		&comic.OriginLanguage,
		&comic.Year,
		&comic.ViewCount,
		&comic.FollowCount,
		&comic.RatingAvg,
		&comic.RatingBayesian,
		&comic.RatingCount,
		&comic.IsLocked,
		&comic.CreatedAt,
		&comic.UpdatedAt,
		&comic.DeletedAt,
		&comic.Links,
		&tagsJSON,
	)

	// Result Validation and Error Handling Strategy
	// Safely checks for empty result sets to bubble up native Application Errors
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("comic")
		}
		return nil, fmt.Errorf("postgres: failed to find comic by id: %w", err)
	}

	// Dynamic Tag Hydration and Unmarshalling
	// Deserializing the batched postgres JSON bytes back into Domain Tag models
	if err := json.Unmarshal(tagsJSON, &comic.Tags); err != nil {
		return nil, fmt.Errorf("postgres: failed to unmarshal tags: %w", err)
	}

	// Returning the validated pipeline struct
	return comic, nil
}

/*
FindBySlug retrieves a comic record securely using its unique SEO URL slug.

Description: Used primarily for public/SEO discovery where the application's internal
UUID is not present in the frontend URL schema. Operates on indexing logic similarly to
FindByID, continuing to utilize Postgres json_agg sub-queries to optimize tag hydration
overhead intelligently during page discovery tracking requests.

Parameters:
  - context: context.Context lifecycle and timeout boundaries
  - slug: string human-readable URL-compliant explicit path identifier

Returns:
  - *Comic: Completely hydrated domain entity containing relation mappings natively
  - error: Generates apperr.NotFound seamlessly on invalid URL mapping parameters
*/
func (repository *comicRepository) FindBySlug(context context.Context, slug string) (*Comic, error) {

	// SEO Lookup Query Initialization with Native Sub-Query Linking
	// Uses the slug column efficiently to fetch core fields while actively coalescing the tags array.
	query := fmt.Sprintf(`
		SELECT 
			c.%s, c.%s, c.%s, c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, c.%s, 
			c.%s, c.%s, c.%s, c.%s, c.%s,
			c.%s,
			COALESCE((
				SELECT json_agg(json_build_object('id', t.%s, 'name', t.%s, 'slug', t.%s))
				FROM %s t
				JOIN %s ct ON t.%s = ct.%s
				WHERE ct.%s = c.%s
			), '[]') as tags
		FROM %s c
		WHERE c.%s = $1 AND c.%s IS NULL
	`,
		schema.CoreComic.ID,
		schema.CoreComic.Title,
		schema.CoreComic.AltTitle,
		schema.CoreComic.Slug,
		schema.CoreComic.Description,
		schema.CoreComic.CoverURL,
		schema.CoreComic.Status,
		schema.CoreComic.ContentRating,
		schema.CoreComic.Demographic,
		schema.CoreComic.DefaultReadMode,
		schema.CoreComic.OriginLanguage,
		schema.CoreComic.Year,
		schema.CoreComic.ViewCount,
		schema.CoreComic.FollowCount,
		schema.CoreComic.RatingAvg,
		schema.CoreComic.RatingBayesian,
		schema.CoreComic.RatingCount,
		schema.CoreComic.IsLocked,
		schema.CoreComic.CreatedAt,
		schema.CoreComic.UpdatedAt,
		schema.CoreComic.DeletedAt,
		schema.CoreComic.Links,
		schema.RefTag.ID,
		schema.RefTag.Name,
		schema.RefTag.Slug,
		schema.RefTag.Table, schema.ComicTag.Table, schema.RefTag.ID, schema.ComicTag.TagID,
		schema.ComicTag.ComicID, schema.CoreComic.ID,
		schema.CoreComic.Table,
		schema.CoreComic.Slug, schema.CoreComic.DeletedAt,
	)

	// Struct Initialization and Target Query Execution
	// Maps standard variables robustly for database target binding.
	comic := &Comic{}
	var tagsJSON []byte

	// Executing the query and parsing multiple columns into the struct via references.
	err := repository.pool.QueryRow(context, query, slug).Scan(
		&comic.ID,
		&comic.Title,
		&comic.TitleAlt,
		&comic.Slug,
		&comic.Synopsis,
		&comic.CoverURL,
		&comic.Status,
		&comic.ContentRating,
		&comic.Demographic,
		&comic.DefaultReadMode,
		&comic.OriginLanguage,
		&comic.Year,
		&comic.ViewCount,
		&comic.FollowCount,
		&comic.RatingAvg,
		&comic.RatingBayesian,
		&comic.RatingCount,
		&comic.IsLocked,
		&comic.CreatedAt,
		&comic.UpdatedAt,
		&comic.DeletedAt,
		&comic.Links,
		&tagsJSON,
	)

	// SEO Validation and Failure Response Mapping
	// Returns a clean 404 domain error explicitly avoiding database leakages.
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("comic_slug")
		}
		return nil, fmt.Errorf("postgres: failed to find comic by slug: %w", err)
	}

	// JSON Aggregation Deserialization Strategy
	// Transforms the raw Postgres JSON byte array safely back into the Go slice architecture.
	if err := json.Unmarshal(tagsJSON, &comic.Tags); err != nil {
		return nil, fmt.Errorf("postgres: failed to unmarshal tags: %w", err)
	}

	// Returning the fully hydrated pointer entity correctly.
	return comic, nil
}

/*
Create persists a new comic entity and all its associated junction table links.

Description: Executes the insertion within a single ACID-compliant PostgreSQL transaction.
This guarantees that if the insertion of the core record or any of the junction links
(tags, authors, artists) fails due to constraints or network issues, the entire operation
is rolled back safely. This explicitly prevents orphaned associations and partial saves.

Parameters:
  - context: context.Context for request scoping and database timeout tracking
  - c: *Comic (The domain entity containing core metadata and junction ID arrays)

Returns:
  - error: Returns nil on a successfully committed sequence or explicitly reports SQL flaws
*/
func (repository *comicRepository) Create(context context.Context, comic *Comic) error {

	// Transaction Context Instantiation
	// Establish an isolated database transaction to safely execute multiple interrelated queries.
	transaction, err := repository.pool.Begin(context)
	if err != nil {
		return fmt.Errorf("postgres: failed to begin transaction: %w", err)
	}

	// Defer Transaction State Reversal
	// Ensures that uncommitted network handles are safely reclaimed if application logic panics.
	defer transaction.Rollback(context)

	// Core Relational Schema Insertion Blueprint
	// Specifies explicit mapping bindings for the root Comic entity initialization.
	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, %s, %s, %s, %s, %s, %s, 
			%s, %s, %s, %s, %s, %s
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		schema.CoreComic.Table,
		schema.CoreComic.ID, schema.CoreComic.Title, schema.CoreComic.AltTitle, schema.CoreComic.Slug, schema.CoreComic.Description, schema.CoreComic.CoverURL, schema.CoreComic.Status,
		schema.CoreComic.ContentRating, schema.CoreComic.Demographic, schema.CoreComic.DefaultReadMode, schema.CoreComic.OriginLanguage, schema.CoreComic.Year, schema.CoreComic.Links,
	)

	// Transaction Execution Dispatch
	// Streams the root parameters into the database explicitly preventing SQL injection boundaries.
	_, err = transaction.Exec(context, query,
		comic.ID,
		comic.Title,
		comic.TitleAlt,
		comic.Slug,
		comic.Synopsis,
		comic.CoverURL,
		comic.Status,
		comic.ContentRating,
		comic.Demographic,
		comic.DefaultReadMode,
		comic.OriginLanguage,
		comic.Year,
		comic.Links,
	)

	// Check for errors during the execution of the query
	if err != nil {
		return fmt.Errorf("postgres: failed to create comic: %w", err)
	}

	// Node Association Synchronizations (Tags)
	// Triggers batch updates mapping many-to-many relationship structures to the initialized row.
	if len(comic.TagIDs) > 0 {
		if err := repository.updateJunction(context, transaction, schema.ComicTag.Table, schema.ComicTag.ComicID, schema.ComicTag.TagID, comic.ID, comic.TagIDs); err != nil {
			return err
		}
	}

	// Node Association Synchronizations (Authors)
	if len(comic.AuthorIDs) > 0 {
		if err := repository.updateJunction(context, transaction, schema.ComicAuthor.Table, schema.ComicAuthor.ComicID, schema.ComicAuthor.AuthorID, comic.ID, comic.AuthorIDs); err != nil {
			return err
		}
	}

	// Node Association Synchronizations (Artists)
	if len(comic.ArtistIDs) > 0 {
		if err := repository.updateJunction(context, transaction, schema.ComicArtist.Table, schema.ComicArtist.ComicID, schema.ComicArtist.ArtistID, comic.ID, comic.ArtistIDs); err != nil {
			return err
		}
	}

	// Final Persistence Validation Strategy
	// Permanently commits the transaction sequence and releases resources safely.
	if err := transaction.Commit(context); err != nil {
		return fmt.Errorf("postgres: failed to commit create transaction: %w", err)
	}

	return nil
}

/*
Update persists metadata modifications to an existing comic record securely.

Description: Utilizes a dynamic SQL strings.Builder to robustly construct a
PATCH-style partial update query. It systematically checks which fields are
populated in the entity and appends them to the SET block dynamically,
optimizing database writes. After the core record is updated, it completely
replaces all junction associations (Tags, Authors, Artists) to maintain
1-to-1 sync mappings safely inside an isolated transactional boundary.

Parameters:
  - context: context.Context handling the database operation lifecycle
  - c: *Comic (Target domain structure containing the targeted UUID and updated fields)

Returns:
  - error: Returns apperr.NotFound if the target record does not exist, or native execution errors
*/
func (repository *comicRepository) Update(context context.Context, comic *Comic) error {

	// Dynamic Query Configuration Pipeline Establishment
	// Initializing buffer builder constructs for the dynamic PATCH update cleanly
	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf("UPDATE %s SET %s = NOW()", schema.CoreComic.Table, schema.CoreComic.UpdatedAt))

	// Initialize query indexing argument counters to manually track positional variables
	var args []any
	argID := 1

	// Parameterization Sequence Generation and Validation Handling
	// Applying variable states individually. This cleanly avoids overwriting existing DB columns with zero values.
	if comic.Title != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Title, argID))
		args = append(args, comic.Title)
		argID++
	}

	// TitleAlt
	if len(comic.TitleAlt) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.AltTitle, argID))
		args = append(args, comic.TitleAlt)
		argID++
	}

	// Slug
	if comic.Slug != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Slug, argID))
		args = append(args, comic.Slug)
		argID++
	}

	// Synopsis
	if comic.Synopsis != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Description, argID))
		args = append(args, comic.Synopsis)
		argID++
	}

	// CoverURL
	if comic.CoverURL != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.CoverURL, argID))
		args = append(args, comic.CoverURL)
		argID++
	}

	// Status
	if comic.Status != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Status, argID))
		args = append(args, comic.Status)
		argID++
	}

	// ContentRating
	if comic.ContentRating != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.ContentRating, argID))
		args = append(args, comic.ContentRating)
		argID++
	}

	// Demographic
	if comic.Demographic != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Demographic, argID))
		args = append(args, comic.Demographic)
		argID++
	}

	// DefaultReadMode
	if comic.DefaultReadMode != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.DefaultReadMode, argID))
		args = append(args, comic.DefaultReadMode)
		argID++
	}

	// OriginLanguage
	if comic.OriginLanguage != "" {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.OriginLanguage, argID))
		args = append(args, comic.OriginLanguage)
		argID++
	}

	// Year
	if comic.Year != nil {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Year, argID))
		args = append(args, *comic.Year)
		argID++
	}

	// Links
	if comic.Links != nil {
		queryBuilder.WriteString(fmt.Sprintf(", %s = $%d", schema.CoreComic.Links, argID))
		args = append(args, comic.Links)
		argID++
	}

	// Targeted Where Constraint Execution Assembly
	// Explicitly bounds the update range to a single primary key and avoids targeting soft-deleted rows.
	queryBuilder.WriteString(fmt.Sprintf(" WHERE %s = $%d AND %s IS NULL", schema.CoreComic.ID, argID, schema.CoreComic.DeletedAt))
	args = append(args, comic.ID)

	// Atomic Transaction Boundary State Setup
	// Instantiates an isolated execution context so junction rewrites rollback if the core query fails.
	transaction, err := repository.pool.Begin(context)
	if err != nil {
		return fmt.Errorf("postgres: update transaction begin failed: %w", err)
	}

	// Error Fallback Rollback Deferral Setup
	// Safely ensures database handles are released if a panic or unexpected closure occurs.
	defer transaction.Rollback(context)

	// Primary Structural Record Integrity Application
	// Commits the organically constructed builder schema locally.
	response, err := transaction.Exec(context, queryBuilder.String(), args...)
	if err != nil {
		return fmt.Errorf("postgres: failed to update comic: %w", err)
	}

	// Missing Entity Constraint Tracking Validation
	// Correctly identifies updates targeting missing or deleted rows and triggers the domain 404 block.
	if response.RowsAffected() == 0 {
		return apperr.NotFound("comic")
	}

	// Graph Synchronization Array Alignment (Tags)
	// Fully wipes and replaces associated relations to safely align the DB array with the frontend payload array.
	if comic.TagIDs != nil {
		if err := repository.updateJunction(context, transaction, schema.ComicTag.Table, schema.ComicTag.ComicID, schema.ComicTag.TagID, comic.ID, comic.TagIDs); err != nil {
			return err
		}
	}

	// Graph Synchronization Array Alignment (Authors)
	if comic.AuthorIDs != nil {
		if err := repository.updateJunction(context, transaction, schema.ComicAuthor.Table, schema.ComicAuthor.ComicID, schema.ComicAuthor.AuthorID, comic.ID, comic.AuthorIDs); err != nil {
			return err
		}
	}

	// Graph Synchronization Array Alignment (Artists)
	if comic.ArtistIDs != nil {
		if err := repository.updateJunction(context, transaction, schema.ComicArtist.Table, schema.ComicArtist.ComicID, schema.ComicArtist.ArtistID, comic.ID, comic.ArtistIDs); err != nil {
			return err
		}
	}

	// Commit Transaction State Logically
	// Persists changes irreversibly after validating the primary table and multi-dimensional junction links correctly.
	if err := transaction.Commit(context); err != nil {
		return fmt.Errorf("postgres: update transaction commit failed: %w", err)
	}

	return nil
}

/*
updateJunction synchronizes many-to-many associations.

Description: Implements a "Clear and Insert" bulk execution strategy for junction tables.
First, it gracefully flushes all mappings associated with a single core root ID to prevent
duplication errors. Then, it utilizes the native `pgx.Batch` execution pipeline to queue
all new relationships natively. This avoids the overhead of executing individual insert queries
and bounds multiple network trips into a highly optimized database sequence.

Parameters:
  - context: context.Context lifecycle mapping
  - tx: pgx.Tx (The actively executed Database Transaction boundary)
  - table: string (The physical fully-qualified table name like "core.comictag")
  - idCol: string (The name of the root relational column, e.g., "comicid")
  - valCol: string (The name of the targeted binding column, e.g., "tagid")
  - id: string (The UUID of the parent referencing entity)
  - vals: []int (The array of foreign keys to safely map to the parent)

Returns:
  - error: Structural execution constraints or failure states
*/
func (repository *comicRepository) updateJunction(context context.Context, transaction pgx.Tx, table, idCol, valCol, id string, vals []int) error {

	// Record Deletion Phase
	// Constructing the physical statement dynamically, cleaning previous entries safely
	delQuery := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, idCol)
	if _, err := transaction.Exec(context, delQuery, id); err != nil {
		return fmt.Errorf("postgres: failed to clear %s: %w", table, err)
	}

	// Empty Array Check
	// Pre-conditions evaluate returning cleanly if no foreign mapping dependencies exist
	if len(vals) == 0 {
		return nil
	}

	// Batch Execution Setup
	// Configures the dynamic INSERT execution mapping to seamlessly queue values iteratively
	insQuery := fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES ($1, $2)", table, idCol, valCol)
	batch := &pgx.Batch{}
	for _, value := range vals {
		batch.Queue(insQuery, id, value)
	}

	// Batch Dispatch
	// Commits the structural mapping strings cleanly under the root payload pipeline sequentially
	response := transaction.SendBatch(context, batch)
	if err := response.Close(); err != nil {
		return fmt.Errorf("postgres: failed to batch insert into %s: %w", table, err)
	}

	return nil
}

/*
SoftDelete hides a comic without physical row removal.

Description: Safely modifies the functional row status by stamping the `deletedat`
column with the database engine's current timestamp natively (`NOW()`). Because
almost all primary application queries feature a global `WHERE deletedat IS NULL`
requirement, this safely scopes it out of user discovery seamlessly.

Parameters:
  - context: context.Context lifecycle and network tracking
  - id: string target unique identifier securely mapped

Returns:
  - error: apperr.NotFound if missing or already deleted, otherwise execution errors
*/
func (repository *comicRepository) SoftDelete(context context.Context, id string) error {

	// Soft Deletion Query Definition
	// Constructs a direct update payload bypassing full record mappings physically.
	query := fmt.Sprintf("UPDATE %s SET %s = NOW() WHERE %s = $1", schema.CoreComic.Table, schema.CoreComic.DeletedAt, schema.CoreComic.ID)

	// Direct Execution Logic
	// Deploys the un-queued query string securely via the native execution pool efficiently
	result, err := repository.pool.Exec(context, query, id)
	if err != nil {
		return fmt.Errorf("postgres: failed to delete comic: %w", err)
	}

	// Structural Modification Validation
	// Determines if rows evaluated the condition bounds reliably handling 404 responses gracefully
	if result.RowsAffected() == 0 {
		return apperr.NotFound("comic")
	}

	return nil
}

/*
IncrementViewCount performs a thread-safe counter update.

Description: Rather than retrieving a target comic, calculating the new popularity metric
algorithmically, and triggering a full entity update, this method simply tells the database
engine to natively apply numeric additions to the column directly. This ensures the integrity
of highly transient interaction metrics during highly concurrent events structurally.

Parameters:
  - context: context.Context native boundaries explicitly mapping bounds
  - id: string specifying the UUID constraint
  - delta: int64 the explicit counter jump parameter, usually 1

Returns:
  - error: Execution failures
*/
func (repository *comicRepository) IncrementViewCount(context context.Context, id string, delta int64) error {

	// Direct Update Definition
	// Assigns the native plus operator mathematically directly on the SQL context execution accurately
	query := fmt.Sprintf("UPDATE %s SET %s = %s + $1 WHERE %s = $2", schema.CoreComic.Table, schema.CoreComic.ViewCount, schema.CoreComic.ViewCount, schema.CoreComic.ID)

	// Contextual Processing Operations
	_, err := repository.pool.Exec(context, query, delta, id)
	if err != nil {
		return fmt.Errorf("postgres: failed to increment view count: %w", err)
	}

	return nil
}
