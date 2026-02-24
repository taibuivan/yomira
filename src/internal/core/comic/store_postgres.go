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
	"github.com/taibuivan/yomira/internal/platform/dberr"
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

// chapterRepository implements the [ChapterRepository] interface using pgx.
type chapterRepository struct {
	pool *pgxpool.Pool
}

// NewChapterRepository constructs a PostgreSQL backed chapter store.
func NewChapterRepository(pool *pgxpool.Pool) ChapterRepository {
	return &chapterRepository{pool: pool}
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
	queryBuilder.WriteString(`
		SELECT 
			c.id, c.title, c.titlealt, c.slug, c.description, c.coverurl, c.status, 
			c.contentrating, c.demographic, c.defaultreadmode, c.originlanguage, 
			c.year, c.viewcount, c.followcount, c.ratingavg, c.ratingbayesian, 
			c.ratingcount, c.islocked, c.createdat, c.updatedat, c.deletedat,
			COUNT(*) OVER() AS total_count,
			COALESCE((
				SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'slug', t.slug))
				FROM core.tag t
				JOIN core.comictag ct ON t.id = ct.tagid
				WHERE ct.comicid = c.id
			), '[]') as tags
		FROM core.comic c
		WHERE c.deletedat IS NULL
	`)

	// Apply Filters (Dynamic WHERE clause construction)
	if len(filter.Status) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.status = ANY($%d)", argID))
		args = append(args, filter.Status)
		argID++
	}

	// Content Rating Filtering
	if len(filter.ContentRating) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.contentrating = ANY($%d)", argID))
		args = append(args, filter.ContentRating)
		argID++
	}

	// Demographic Filtering
	if len(filter.Demographic) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.demographic = ANY($%d)", argID))
		args = append(args, filter.Demographic)
		argID++
	}

	// Origin Language Filtering
	if len(filter.OriginLanguage) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.originlanguage = ANY($%d)", argID))
		args = append(args, filter.OriginLanguage)
		argID++
	}

	// Year Filtering
	if filter.Year != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.year = $%d", argID))
		args = append(args, *filter.Year)
		argID++
	}

	// Search Query Filtering
	if filter.Query != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND c.searchvector @@ websearch_to_tsquery('simple', unaccent($%d))", argID))
		args = append(args, filter.Query)
		argID++
	}

	// Tag Filtering (AND logic for included, NOT EXISTS for excluded)
	if len(filter.IncludedTags) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND $%d::int[] <@ (SELECT array_agg(tagid) FROM core.comictag WHERE comicid = c.id)`, argID))
		args = append(args, filter.IncludedTags)
		argID++
	}

	// Excluded Tags Filtering
	if len(filter.ExcludedTags) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND NOT ($%d::int[] && (SELECT array_agg(tagid) FROM core.comictag WHERE comicid = c.id))`, argID))
		args = append(args, filter.ExcludedTags)
		argID++
	}

	// Author/Artist Filtering
	// Included Authors Filtering
	if len(filter.IncludedAuthors) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND $%d::int[] && (SELECT array_agg(authorid) FROM core.comicauthor WHERE comicid = c.id)`, argID))
		args = append(args, filter.IncludedAuthors)
		argID++
	}

	// Included Artists Filtering
	if len(filter.IncludedArtists) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(` AND $%d::int[] && (SELECT array_agg(artistid) FROM core.comicartist WHERE comicid = c.id)`, argID))
		args = append(args, filter.IncludedArtists)
		argID++
	}

	// Apply Sorting Logic
	sort := "c.createdat" // default
	switch filter.Sort {
	// Popularity
	case "popular":
		sort = "c.viewcount"
	// Rating
	case "rating":
		sort = "c.ratingbayesian"
	// Follow Count
	case "followcount":
		sort = "c.followcount"
	// Alphabetical Order
	case "az":
		sort = "c.title"
	case "za":
		sort = "c.title"
	// Latest
	case "latest":
		sort = "c.createdat"
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
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s, c.id DESC", sort, sortDir))

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
	query := `
		SELECT 
			c.id, c.title, c.titlealt, c.slug, c.description, c.coverurl, c.status, 
			c.contentrating, c.demographic, c.defaultreadmode, c.originlanguage, 
			c.year, c.viewcount, c.followcount, c.ratingavg, c.ratingbayesian, 
			c.ratingcount, c.islocked, c.createdat, c.updatedat, c.deletedat,
			c.links,
			COALESCE((
				SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'slug', t.slug))
				FROM core.tag t
				JOIN core.comictag ct ON t.id = ct.tagid
				WHERE ct.comicid = c.id
			), '[]') as tags
		FROM core.comic c
		WHERE c.id = $1 AND c.deletedat IS NULL
	`

	// Variable Initialization and Record Scanning Pipeline
	// Allocates the local comic instance and a raw byte slice for dynamic JSON processing
	comic := &Comic{}
	var tagsJSON []byte

	// Execute the structured query on the connection pool and map columns dynamically
	err := repository.pool.QueryRow(context, query, id).Scan(
		&comic.ID, &comic.Title, &comic.TitleAlt, &comic.Slug, &comic.Synopsis, &comic.CoverURL, &comic.Status,
		&comic.ContentRating, &comic.Demographic, &comic.DefaultReadMode, &comic.OriginLanguage,
		&comic.Year, &comic.ViewCount, &comic.FollowCount, &comic.RatingAvg, &comic.RatingBayesian,
		&comic.RatingCount, &comic.IsLocked, &comic.CreatedAt, &comic.UpdatedAt, &comic.DeletedAt,
		&comic.Links, &tagsJSON,
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
	query := `
		SELECT 
			c.id, c.title, c.titlealt, c.slug, c.description, c.coverurl, c.status, 
			c.contentrating, c.demographic, c.defaultreadmode, c.originlanguage, 
			c.year, c.viewcount, c.followcount, c.ratingavg, c.ratingbayesian, 
			c.ratingcount, c.islocked, c.createdat, c.updatedat, c.deletedat,
			c.links,
			COALESCE((
				SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'slug', t.slug))
				FROM core.tag t
				JOIN core.comictag ct ON t.id = ct.tagid
				WHERE ct.comicid = c.id
			), '[]') as tags
		FROM core.comic c
		WHERE c.slug = $1 AND c.deletedat IS NULL
	`

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
	query := `
		INSERT INTO core.comic (
			id, title, titlealt, slug, description, coverurl, status, 
			contentrating, demographic, defaultreadmode, originlanguage, year, links
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

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
		if err := repository.updateJunction(context, transaction, "core.comictag", "comicid", "tagid", comic.ID, comic.TagIDs); err != nil {
			return err
		}
	}

	// Node Association Synchronizations (Authors)
	if len(comic.AuthorIDs) > 0 {
		if err := repository.updateJunction(context, transaction, "core.comicauthor", "comicid", "authorid", comic.ID, comic.AuthorIDs); err != nil {
			return err
		}
	}

	// Node Association Synchronizations (Artists)
	if len(comic.ArtistIDs) > 0 {
		if err := repository.updateJunction(context, transaction, "core.comicartist", "comicid", "artistid", comic.ID, comic.ArtistIDs); err != nil {
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
	queryBuilder.WriteString("UPDATE core.comic SET updatedat = NOW()")

	// Initialize query indexing argument counters to manually track positional variables
	var args []any
	argID := 1

	// Parameterization Sequence Generation and Validation Handling
	// Applying variable states individually. This cleanly avoids overwriting existing DB columns with zero values.
	if comic.Title != "" {
		queryBuilder.WriteString(fmt.Sprintf(", title = $%d", argID))
		args = append(args, comic.Title)
		argID++
	}

	// TitleAlt
	if len(comic.TitleAlt) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(", titlealt = $%d", argID))
		args = append(args, comic.TitleAlt)
		argID++
	}

	// Slug
	if comic.Slug != "" {
		queryBuilder.WriteString(fmt.Sprintf(", slug = $%d", argID))
		args = append(args, comic.Slug)
		argID++
	}

	// Synopsis
	if comic.Synopsis != "" {
		queryBuilder.WriteString(fmt.Sprintf(", description = $%d", argID))
		args = append(args, comic.Synopsis)
		argID++
	}

	// CoverURL
	if comic.CoverURL != "" {
		queryBuilder.WriteString(fmt.Sprintf(", coverurl = $%d", argID))
		args = append(args, comic.CoverURL)
		argID++
	}

	// Status
	if comic.Status != "" {
		queryBuilder.WriteString(fmt.Sprintf(", status = $%d", argID))
		args = append(args, comic.Status)
		argID++
	}

	// ContentRating
	if comic.ContentRating != "" {
		queryBuilder.WriteString(fmt.Sprintf(", contentrating = $%d", argID))
		args = append(args, comic.ContentRating)
		argID++
	}

	// Demographic
	if comic.Demographic != "" {
		queryBuilder.WriteString(fmt.Sprintf(", demographic = $%d", argID))
		args = append(args, comic.Demographic)
		argID++
	}

	// DefaultReadMode
	if comic.DefaultReadMode != "" {
		queryBuilder.WriteString(fmt.Sprintf(", defaultreadmode = $%d", argID))
		args = append(args, comic.DefaultReadMode)
		argID++
	}

	// OriginLanguage
	if comic.OriginLanguage != "" {
		queryBuilder.WriteString(fmt.Sprintf(", originlanguage = $%d", argID))
		args = append(args, comic.OriginLanguage)
		argID++
	}

	// Year
	if comic.Year != nil {
		queryBuilder.WriteString(fmt.Sprintf(", year = $%d", argID))
		args = append(args, *comic.Year)
		argID++
	}

	// Links
	if comic.Links != nil {
		queryBuilder.WriteString(fmt.Sprintf(", links = $%d", argID))
		args = append(args, comic.Links)
		argID++
	}

	// Targeted Where Constraint Execution Assembly
	// Explicitly bounds the update range to a single primary key and avoids targeting soft-deleted rows.
	queryBuilder.WriteString(fmt.Sprintf(" WHERE id = $%d AND deletedat IS NULL", argID))
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
		if err := repository.updateJunction(context, transaction, "core.comictag", "comicid", "tagid", comic.ID, comic.TagIDs); err != nil {
			return err
		}
	}

	// Graph Synchronization Array Alignment (Authors)
	if comic.AuthorIDs != nil {
		if err := repository.updateJunction(context, transaction, "core.comicauthor", "comicid", "authorid", comic.ID, comic.AuthorIDs); err != nil {
			return err
		}
	}

	// Graph Synchronization Array Alignment (Artists)
	if comic.ArtistIDs != nil {
		if err := repository.updateJunction(context, transaction, "core.comicartist", "comicid", "artistid", comic.ID, comic.ArtistIDs); err != nil {
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
	query := `UPDATE core.comic SET deletedat = NOW() WHERE id = $1`

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
	query := `UPDATE core.comic SET viewcount = viewcount + $1 WHERE id = $2`

	// Contextual Processing Operations
	_, err := repository.pool.Exec(context, query, delta, id)
	if err != nil {
		return fmt.Errorf("postgres: failed to increment view count: %w", err)
	}

	return nil
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

	queryBuilder.WriteString(`
		SELECT 
			c.id, c.comicid, c.chapternumber, c.title, l.code as language,
			c.scanlationgroupid, c.publishedat, c.createdat, c.updatedat, c.deletedat,
			COUNT(*) OVER() AS total_count
		FROM core.chapter c
		JOIN core.language l ON c.languageid = l.id
		WHERE c.comicid = $1 AND c.deletedat IS NULL
	`)
	args = append(args, comicID)
	argID++

	// Language filter injection
	if filter.Language != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND l.code = $%d", argID))
		args = append(args, filter.Language)
		argID++
	}

	// Ordering and pagination limits
	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY c.chapternumber %s", sortDir))
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
			&chapter.ID, &chapter.ComicID, &chapter.Number, &chapter.Title, &chapter.Language,
			&scangroupID, &chapter.PublishedAt, &chapter.CreatedAt, &chapter.UpdatedAt, &chapter.DeletedAt,
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
	query := `
		SELECT 
			c.id, c.comicid, c.chapternumber, c.title, l.code as language,
			c.scanlationgroupid, c.externalurl, c.islocked, c.publishedat, 
			c.createdat, c.updatedat, c.deletedat
		FROM core.chapter c
		JOIN core.language l ON c.languageid = l.id
		WHERE c.id = $1 AND c.deletedat IS NULL
	`

	// Initialize chapter entity mapping targets
	var chapter Chapter
	var scangroupID *string

	// Execute query and extract mapping parameters
	err := repository.pool.QueryRow(context, query, id).Scan(
		&chapter.ID, &chapter.ComicID, &chapter.Number, &chapter.Title, &chapter.Language,
		&scangroupID, &chapter.ExternalURL, &chapter.IsLocked, &chapter.PublishedAt,
		&chapter.CreatedAt, &chapter.UpdatedAt, &chapter.DeletedAt,
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
	query := `
		INSERT INTO core.chapter (
			id, comicid, languageid, chapternumber, title, 
			externalurl, islocked, publishedat
		) VALUES (
			$1, $2, (SELECT id FROM core.language WHERE code = $3), $4, $5, $6, $7, $8
		)
	`

	// Stream mapping targets executing directly against connection pool streams
	_, err := repository.pool.Exec(context, query,
		chapter.ID, chapter.ComicID, chapter.Language, chapter.Number, chapter.Title,
		chapter.ExternalURL, chapter.IsLocked, chapter.PublishedAt,
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
	query := `
		UPDATE core.chapter
		SET 
			chapternumber = $1, title = $2, 
			languageid = (SELECT id FROM core.language WHERE code = $3),
			externalurl = $4, islocked = $5, publishedat = $6, updatedat = NOW()
		WHERE id = $7 AND deletedat IS NULL
	`

	// Execute record update
	result, err := repository.pool.Exec(context, query,
		chapter.Number, chapter.Title, chapter.Language, chapter.ExternalURL,
		chapter.IsLocked, chapter.PublishedAt, chapter.ID,
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
	query := `UPDATE core.chapter SET deletedat = NOW() WHERE id = $1`

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
	query := `
		SELECT id, chapterid, pagenumber, imageurl
		FROM core.page
		WHERE chapterid = $1
		ORDER BY pagenumber ASC
	`

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
		batch.Queue(`
			INSERT INTO core.page (id, chapterid, pagenumber, imageurl)
			VALUES ($1, $2, $3, $4)
		`, p.ID, p.ChapterID, p.PageNumber, p.ImageURL)
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
	query := `UPDATE core.chapter SET viewcount = viewcount + $1 WHERE id = $2`

	_, err := repository.pool.Exec(context, query, delta, id)
	if err != nil {
		return fmt.Errorf("postgres: failed to increment chapter view count: %w", err)
	}

	return nil
}

// # Title & Relation Management

/*
ListTitles returns all alternative titles for a comic.
*/
func (repository *comicRepository) ListTitles(context context.Context, comicID string) ([]*Title, error) {

	// Joint query to resolve language labels
	query := `
		SELECT t.comicid, l.code, t.title
		FROM core.comictitle t
		JOIN core.language l ON t.languageid = l.id
		WHERE t.comicid = $1
	`

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
	query := `
		INSERT INTO core.comictitle (comicid, languageid, title)
		VALUES ($1, (SELECT id FROM core.language WHERE code = $2), $3)
		ON CONFLICT (comicid, languageid) DO UPDATE SET title = EXCLUDED.title
	`

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
	query := `
		DELETE FROM core.comictitle 
		WHERE comicid = $1 AND languageid = (SELECT id FROM core.language WHERE code = $2)
	`
	_, err := repository.pool.Exec(context, query, comicID, langCode)
	return err
}

/*
ListRelations fetches all linked publications.
*/
func (repository *comicRepository) ListRelations(context context.Context, comicID string) ([]*Relation, error) {

	// Relation lookup with title resolution
	query := `
		SELECT r.fromcomicid, r.tocomicid, r.relationtype, c.title
		FROM core.comicrelation r
		JOIN core.comic c ON r.tocomicid = c.id
		WHERE r.fromcomicid = $1
	`

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
	query := `
		INSERT INTO core.comicrelation (fromcomicid, tocomicid, relationtype)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

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
	query := `DELETE FROM core.comicrelation WHERE fromcomicid = $1 AND tocomicid = $2 AND relationtype = $3`

	// Command execution
	_, err := repository.pool.Exec(context, query, fromID, toID, relType)
	return err
}

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
	query := `
		SELECT id, comicid, volume, imageurl, description, createdat
		FROM core.comiccover
		WHERE comicid = $1
		ORDER BY volume ASC NULLS LAST, createdat DESC
	`

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
			&cover.ID, &cover.ComicID, &cover.Volume,
			&cover.ImageURL, &cover.Description, &cover.CreatedAt,
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
	query := `
		INSERT INTO core.comiccover (id, comicid, volume, imageurl, description)
		VALUES ($1, $2, $3, $4, $5)
	`

	// Execution
	_, err := repository.pool.Exec(context, query,
		cover.ID, cover.ComicID, cover.Volume,
		cover.ImageURL, cover.Description,
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
	query := `DELETE FROM core.comiccover WHERE id = $1`
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
	queryBuilder.WriteString(`
		SELECT id, comicid, uploaderid, imageurl, isapproved, createdat
		FROM core.comicart
		WHERE comicid = $1
	`)

	args := []any{comicID}

	// Status Filter Application
	if onlyApproved {
		queryBuilder.WriteString(" AND isapproved = true")
	}

	queryBuilder.WriteString(" ORDER BY createdat DESC")

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
			&art.ID, &art.ComicID, &art.UploaderID,
			&art.ImageURL, &art.IsApproved, &art.CreatedAt,
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
	query := `
		INSERT INTO core.comicart (id, comicid, uploaderid, imageurl, isapproved)
		VALUES ($1, $2, $3, $4, $5)
	`

	// Contextual Execute
	_, err := repository.pool.Exec(context, query,
		art.ID, art.ComicID, art.UploaderID,
		art.ImageURL, art.IsApproved,
	)

	return dberr.Wrap(err, "add_art")
}

/*
DeleteArt removes a gallery image.
*/
func (repository *comicRepository) DeleteArt(context context.Context, id string) error {
	query := `DELETE FROM core.comicart WHERE id = $1`
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
	query := `UPDATE core.comicart SET isapproved = $1 WHERE id = $2`

	// Batch pool execution
	_, err := repository.pool.Exec(context, query, approved, id)

	return dberr.Wrap(err, "approve_art")
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
	query := `
		INSERT INTO core.userread (userid, chapterid)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	// Atomic Execute
	_, err := repository.pool.Exec(context, query, userID, chapterID)

	return dberr.Wrap(err, "mark_as_read")
}
