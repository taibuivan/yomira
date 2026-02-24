// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package reference

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/taibuivan/yomira/internal/platform/dberr"
)

// PostgresRepository implements [Repository] using a pgxpool.
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository returns a fully wired postgres implementation.
func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

/*
ListLanguages retrieves all supported linguistic locales.

Description: Selects the full set of language records from the core schema,
ordered by their primary identifier.

Parameters:
  - context: context.Context

Returns:
  - []*Language: Collection of localized metadata
  - error: Database execution or scanning errors
*/
func (repository *PostgresRepository) ListLanguages(context context.Context) ([]*Language, error) {

	// Define language retrieval query
	const query = `
		SELECT id, code, name, nativename
		FROM core.language
		ORDER BY id ASC;
	`

	// Execute retrieval against connection pool
	rows, err := repository.db.Query(context, query)
	if err != nil {
		return nil, dberr.Wrap(err, "list_languages")
	}
	defer rows.Close()

	// Iterate results and hydrate entity slice
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

/*
GetLanguageByCode fetches a single locale by its ISO-639-1 code.

Description: Performs a direct lookup on the language table using its unique code.

Parameters:
  - context: context.Context
  - code: string (ISO-639-1)

Returns:
  - *Language: The hydrated locale entity
  - error: Not found or execution errors
*/
func (repository *PostgresRepository) GetLanguageByCode(context context.Context, code string) (*Language, error) {

	// Prepare single-row selection
	const query = `
		SELECT id, code, name, nativename
		FROM core.language
		WHERE code = $1;
	`

	// Execute query and scan directly into entity
	l := &Language{}
	err := repository.db.QueryRow(context, query, code).Scan(&l.ID, &l.Code, &l.Name, &l.NativeName)

	return l, dberr.Wrap(err, "get_language")
}

/*
ListTags retrieves all tag groups and their nested tags.

Description: executes two optimized queries to fetch groups and tags separately,
reconstructing the hierarchical tree structure in memory.

Parameters:
  - context: context.Context

Returns:
  - []*TagGroup: Hierarchical collection of tags organized by group
  - error: Database retrieval failures
*/
func (repository *PostgresRepository) ListTags(context context.Context) ([]*TagGroup, error) {
	const gQuery = `SELECT id, name, slug, sortorder FROM core.taggroup ORDER BY sortorder ASC`
	const tQuery = `SELECT id, groupid, name, slug, description FROM core.tag ORDER BY name ASC`

	// Retrieve tag groups first
	gRows, err := repository.db.Query(context, gQuery)
	if err != nil {
		return nil, dberr.Wrap(err, "list_tag_groups")
	}
	defer gRows.Close()

	// Hydrate groups and build lookup map
	groups := make([]*TagGroup, 0)
	groupMap := make(map[int]*TagGroup)

	for gRows.Next() {
		g := &TagGroup{Tags: make([]Tag, 0)}
		if err := gRows.Scan(&g.ID, &g.Name, &g.Slug, &g.SortOrder); err != nil {
			return nil, dberr.Wrap(err, "scan_tag_group")
		}
		groups = append(groups, g)
		groupMap[g.ID] = g
	}
	gRows.Close()

	// Retrieve all tags
	tRows, err := repository.db.Query(context, tQuery)
	if err != nil {
		return nil, dberr.Wrap(err, "list_tags")
	}
	defer tRows.Close()

	// Assign tags to their respective parent groups
	for tRows.Next() {
		t := Tag{}
		if err := tRows.Scan(&t.ID, &t.GroupID, &t.Name, &t.Slug, &t.Description); err != nil {
			return nil, dberr.Wrap(err, "scan_tag")
		}

		if grp, ok := groupMap[t.GroupID]; ok {
			grp.Tags = append(grp.Tags, t)
		}
	}

	return groups, nil
}

/*
GetTagByID retrieves a specific tag by its primary key.

Description: Uses a join between tag and taggroup to return a fully hydrated tag object.

Parameters:
  - context: context.Context
  - id: int identifier

Returns:
  - *Tag: Hydrated tag entity with its group
  - error: Database execution or not found errors
*/
func (repository *PostgresRepository) GetTagByID(context context.Context, id int) (*Tag, error) {

	// Define relational query with group joining
	const query = `
		SELECT t.id, t.groupid, t.name, t.slug, t.description,
		       g.id, g.name, g.slug, g.sortorder
		FROM core.tag t
		JOIN core.taggroup g ON t.groupid = g.id
		WHERE t.id = $1
	`
	t := &Tag{}
	g := &TagGroup{}

	// Execute join query and scan parameters
	err := repository.db.QueryRow(context, query, id).Scan(
		&t.ID, &t.GroupID, &t.Name, &t.Slug, &t.Description,
		&g.ID, &g.Name, &g.Slug, &g.SortOrder,
	)

	if err != nil {
		return nil, dberr.Wrap(err, "get_tag_by_id")
	}

	// Connect group and return
	t.Group = g
	return t, nil
}

/*
GetTagBySlug retrieves a tag using its URL identifier.

Description: Resolves a unique slug into its tag entity and parent group.

Parameters:
  - context: context.Context
  - slug: string semantic identifier

Returns:
  - *Tag: Hydrated tag entity
  - error: Retrieval failures
*/
func (repository *PostgresRepository) GetTagBySlug(context context.Context, slug string) (*Tag, error) {

	// Define lookup query
	const query = `
		SELECT t.id, t.groupid, t.name, t.slug, t.description,
		       g.id, g.name, g.slug, g.sortorder
		FROM core.tag t
		JOIN core.taggroup g ON t.groupid = g.id
		WHERE t.slug = $1
	`
	t := &Tag{}
	g := &TagGroup{}

	// Execute scan from pool
	err := repository.db.QueryRow(context, query, slug).Scan(
		&t.ID, &t.GroupID, &t.Name, &t.Slug, &t.Description,
		&g.ID, &g.Name, &g.Slug, &g.SortOrder,
	)

	if err != nil {
		return nil, dberr.Wrap(err, "get_tag_by_slug")
	}

	t.Group = g
	return t, nil
}

/*
ListAuthors retrieves a filtered and paginated list of catalog authors.

Description: Supports name search (ILIKE) on primary and alternative titles,
returning both the entity slice and a total count for pagination metadata.

Parameters:
  - context: context.Context
  - f: ContributorFilter (Search parameters)
  - limit, offset: int (Pagination bounds)

Returns:
  - []*Author: Paginated results
  - int: Total matching count
  - error: Database execution errors
*/
func (repository *PostgresRepository) ListAuthors(context context.Context, f ContributorFilter, limit, offset int) ([]*Author, int, error) {

	// Base queries for selection and counting
	query := `
		SELECT id, name, namealt, bio, imageurl, createdat, updatedat
		FROM core.author
		WHERE deletedat IS NULL
	`
	countQuery := `SELECT count(*) FROM core.author WHERE deletedat IS NULL`

	args := []any{}
	countArgs := []any{}

	// Apply filter parameters
	if f.Query != "" {
		searchTerm := "%" + f.Query + "%"
		query += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		countQuery += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		args = append(args, searchTerm)
		countArgs = append(countArgs, searchTerm)
	}

	// Append ordering and pagination bounds
	query += ` ORDER BY name ASC LIMIT $` + itos(len(args)+1) + ` OFFSET $` + itos(len(args)+2)
	args = append(args, limit, offset)

	// Retrieve total count for metadata
	var total int
	if err := repository.db.QueryRow(context, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, dberr.Wrap(err, "count_authors")
	}

	// Execute paginated selection
	rows, err := repository.db.Query(context, query, args...)
	if err != nil {
		return nil, 0, dberr.Wrap(err, "list_authors")
	}
	defer rows.Close()

	// Hydrate result set
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

/*
GetAuthor retrieves a single author profile by its primary key.

Parameters:
  - context: context.Context
  - id: int identifier

Returns:
  - *Author: Hydrated profile entity
  - error: Domain standard or SQL errors
*/
func (repository *PostgresRepository) GetAuthor(context context.Context, id int) (*Author, error) {

	// Selection query targeting active records
	const query = `
		SELECT id, name, namealt, bio, imageurl, createdat, updatedat
		FROM core.author
		WHERE id = $1 AND deletedat IS NULL
	`
	a := &Author{}

	// Execute scan from pool
	err := repository.db.QueryRow(context, query, id).Scan(
		&a.ID, &a.Name, &a.NameAlt, &a.Bio, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt,
	)

	return a, dberr.Wrap(err, "get_author")
}

/*
CreateAuthor persists a new author record.

Description: Inserts author metadata and returns the auto-generated primary key
and timestamp values.

Parameters:
  - context: context.Context
  - a: *Author (Input entity)

Returns:
  - error: Column constraint violations or execution errors
*/
func (repository *PostgresRepository) CreateAuthor(context context.Context, a *Author) error {

	// Construct insertion statement returning audit fields
	const query = `
		INSERT INTO core.author (name, namealt, bio, imageurl, createdat, updatedat)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, createdat, updatedat
	`

	// Execute and bind generated responses
	err := repository.db.QueryRow(context, query, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)

	return dberr.Wrap(err, "create_author")
}

/*
UpdateAuthor applies modifications to an existing author record.

Description: Performs a targeted update on metadata fields for an active author.

Parameters:
  - context: context.Context
  - a: *Author (Input entity with valid ID)

Returns:
  - error: Update failures or apperr.NotFound
*/
func (repository *PostgresRepository) UpdateAuthor(context context.Context, a *Author) error {

	// Construct update statement with timestamp renewal
	const query = `
		UPDATE core.author
		SET name = $2, namealt = $3, bio = $4, imageurl = $5, updatedat = NOW()
		WHERE id = $1 AND deletedat IS NULL
		RETURNING updatedat
	`

	// Execute update directly in repository
	err := repository.db.QueryRow(context, query, a.ID, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.UpdatedAt)

	return dberr.Wrap(err, "update_author")
}

/*
DeleteAuthor flags an author as logically destroyed.

Description: Soft-deletion behavior using a deletedat timestamp.

Parameters:
  - context: context.Context
  - id: int (Primary key)

Returns:
  - error: Deletion failures or dberr.ErrNotFound
*/
func (repository *PostgresRepository) DeleteAuthor(context context.Context, id int) error {

	// Define soft-deletion SQL
	const query = `UPDATE core.author SET deletedat = NOW() WHERE id = $1 AND deletedat IS NULL`

	// Execute statement
	cmd, err := repository.db.Exec(context, query, id)

	if err != nil {
		return dberr.Wrap(err, "delete_author")
	}

	// Verify target existence
	if cmd.RowsAffected() == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

/*
ListArtists retrieves illustrator metadata from the catalogue.

Description: Selection with trigram-based keyword filtering.

Parameters:
  - context: context.Context
  - f: ContributorFilter
  - limit, offset: int

Returns:
  - []*Artist: Collection of artists
  - int: Total matching count
  - error: Database retrieval failures
*/
func (repository *PostgresRepository) ListArtists(context context.Context, f ContributorFilter, limit, offset int) ([]*Artist, int, error) {

	// Base query configuration
	query := `
		SELECT id, name, namealt, bio, imageurl, createdat, updatedat
		FROM core.artist
		WHERE deletedat IS NULL
	`
	countQuery := `SELECT count(*) FROM core.artist WHERE deletedat IS NULL`

	args := []any{}
	countArgs := []any{}

	// Apply keyword filtering
	if f.Query != "" {
		searchTerm := "%" + f.Query + "%"
		query += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		countQuery += ` AND (name ILIKE $1 OR array_to_string(namealt, ' ') ILIKE $1)`
		args = append(args, searchTerm)
		countArgs = append(countArgs, searchTerm)
	}

	// Setup pagination
	query += ` ORDER BY name ASC LIMIT $` + itos(len(args)+1) + ` OFFSET $` + itos(len(args)+2)
	args = append(args, limit, offset)

	// Fetch statistics
	var total int
	if err := repository.db.QueryRow(context, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, dberr.Wrap(err, "count_artists")
	}

	// Execute data retrieval
	rows, err := repository.db.Query(context, query, args...)
	if err != nil {
		return nil, 0, dberr.Wrap(err, "list_artists")
	}
	defer rows.Close()

	// Parse records
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

/*
GetArtist retrieves an illustrator profile by ID.

Parameters:
  - context: context.Context
  - id: int secondary key

Returns:
  - *Artist: Domain entity mapping
  - error: Postgres execution or mapping failures
*/
func (repository *PostgresRepository) GetArtist(context context.Context, id int) (*Artist, error) {
	const query = `
		SELECT id, name, namealt, bio, imageurl, createdat, updatedat
		FROM core.artist
		WHERE id = $1 AND deletedat IS NULL
	`
	a := &Artist{}
	err := repository.db.QueryRow(context, query, id).Scan(
		&a.ID, &a.Name, &a.NameAlt, &a.Bio, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt,
	)
	return a, dberr.Wrap(err, "get_artist")
}

/*
CreateArtist persists a new illustrator profile.

Parameters:
  - context: context.Context
  - a: *Artist (Entity input)

Returns:
  - error: Database constraint violations
*/
func (repository *PostgresRepository) CreateArtist(context context.Context, a *Artist) error {
	const query = `
		INSERT INTO core.artist (name, namealt, bio, imageurl, createdat, updatedat)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, createdat, updatedat
	`
	err := repository.db.QueryRow(context, query, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	return dberr.Wrap(err, "create_artist")
}

/*
UpdateArtist modifies an active artist record.

Parameters:
  - context: context.Context
  - a: *Artist (Entity with ID)

Returns:
  - error: Persistence failures
*/
func (repository *PostgresRepository) UpdateArtist(context context.Context, a *Artist) error {
	const query = `
		UPDATE core.artist
		SET name = $2, namealt = $3, bio = $4, imageurl = $5, updatedat = NOW()
		WHERE id = $1 AND deletedat IS NULL
		RETURNING updatedat
	`
	err := repository.db.QueryRow(context, query, a.ID, a.Name, a.NameAlt, a.Bio, a.ImageURL).Scan(&a.UpdatedAt)
	return dberr.Wrap(err, "update_artist")
}

/*
DeleteArtist flags an illustrator profile as logically deleted.

Parameters:
  - context: context.Context
  - id: int identifier

Returns:
  - error: Database errors or dberr.ErrNotFound
*/
func (repository *PostgresRepository) DeleteArtist(context context.Context, id int) error {
	const query = `UPDATE core.artist SET deletedat = NOW() WHERE id = $1 AND deletedat IS NULL`
	cmd, err := repository.db.Exec(context, query, id)
	if err != nil {
		return dberr.Wrap(err, "delete_artist")
	}
	if cmd.RowsAffected() == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

// helper for integer array bindings
// itos converts an integer to a string.
// It is used for building dynamic SQL parameter placeholders (e.g., $1, $2).
func itos(i int) string {
	return strconv.Itoa(i)
}
