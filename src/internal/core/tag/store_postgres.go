package tag

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

func (repository *PostgresRepository) ListTags(context context.Context) ([]*TagGroup, error) {
	gQuery := fmt.Sprintf(`SELECT %s, %s, %s, %s FROM %s ORDER BY %s ASC`,
		schema.RefTagGroup.ID, schema.RefTagGroup.Name, schema.RefTagGroup.Slug, schema.RefTagGroup.SortOrder,
		schema.RefTagGroup.Table, schema.RefTagGroup.SortOrder)
	tQuery := fmt.Sprintf(`SELECT %s, %s, %s, %s, %s FROM %s ORDER BY %s ASC`,
		schema.RefTag.ID, schema.RefTag.GroupID, schema.RefTag.Name, schema.RefTag.Slug, schema.RefTag.Description,
		schema.RefTag.Table, schema.RefTag.Name)

	gRows, err := repository.db.Query(context, gQuery)
	if err != nil {
		return nil, dberr.Wrap(err, "list_tag_groups")
	}
	defer gRows.Close()

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

	tRows, err := repository.db.Query(context, tQuery)
	if err != nil {
		return nil, dberr.Wrap(err, "list_tags")
	}
	defer tRows.Close()

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

func (repository *PostgresRepository) GetTagByID(context context.Context, id int) (*Tag, error) {
	query := fmt.Sprintf(`
		SELECT t.%s, t.%s, t.%s, t.%s, t.%s,
		       g.%s, g.%s, g.%s, g.%s
		FROM %s t
		JOIN %s g ON t.%s = g.%s
		WHERE t.%s = $1
	`,
		schema.RefTag.ID, schema.RefTag.GroupID, schema.RefTag.Name, schema.RefTag.Slug, schema.RefTag.Description,
		schema.RefTagGroup.ID, schema.RefTagGroup.Name, schema.RefTagGroup.Slug, schema.RefTagGroup.SortOrder,
		schema.RefTag.Table, schema.RefTagGroup.Table,
		schema.RefTag.GroupID, schema.RefTagGroup.ID, schema.RefTag.ID,
	)
	t := &Tag{}
	g := &TagGroup{}

	err := repository.db.QueryRow(context, query, id).Scan(
		&t.ID, &t.GroupID, &t.Name, &t.Slug, &t.Description,
		&g.ID, &g.Name, &g.Slug, &g.SortOrder,
	)
	if err != nil {
		return nil, dberr.Wrap(err, "get_tag_by_id")
	}

	t.Group = g
	return t, nil
}

func (repository *PostgresRepository) GetTagBySlug(context context.Context, slug string) (*Tag, error) {
	query := fmt.Sprintf(`
		SELECT t.%s, t.%s, t.%s, t.%s, t.%s,
		       g.%s, g.%s, g.%s, g.%s
		FROM %s t
		JOIN %s g ON t.%s = g.%s
		WHERE t.%s = $1
	`,
		schema.RefTag.ID, schema.RefTag.GroupID, schema.RefTag.Name, schema.RefTag.Slug, schema.RefTag.Description,
		schema.RefTagGroup.ID, schema.RefTagGroup.Name, schema.RefTagGroup.Slug, schema.RefTagGroup.SortOrder,
		schema.RefTag.Table, schema.RefTagGroup.Table,
		schema.RefTag.GroupID, schema.RefTagGroup.ID, schema.RefTag.Slug,
	)
	t := &Tag{}
	g := &TagGroup{}

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
