package tag

import "time"

// TagGroup provides a logical grouping for tags to prevent flat list overload.
type TagGroup struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"-"`

	// Tags contains the child tags for this group, populated in hierarchical queries.
	Tags []Tag `json:"tags,omitempty"`
}

// Tag represents a specific categorization attribute applied to a comic.
type Tag struct {
	ID          int       `json:"id"`
	GroupID     int       `json:"group_id,omitempty"`
	Group       *TagGroup `json:"group,omitempty"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"-"`
}
