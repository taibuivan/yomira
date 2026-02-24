/*
Package reference manages the "Master Data" or taxonomic foundations of Yomira.

It handles the lifecycle and retrieval of reference entities that are shared across
multiple comics, ensuring data consistency and enabling rich discovery features.

# Core Responsibility

  - Taxonomy: Manages hierarchical [TagGroup] and [Tag] trees.
  - Localization: Maintains supported [Language] codes (ISO/BCP-47).
  - Authorship: Catalogues [Author] and [Artist] entities with multi-language name support.

This package provides the "Common Language" used by the core catalogue to categorize content.
*/
package reference

import "time"

// # Language Domain

// Language represents a spoken/written language supported by the system.
type Language struct {
	ID         int       `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	NativeName string    `json:"native_name"`
	CreatedAt  time.Time `json:"-"`
}

// # Tag Domain

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

// # Contributor Domain

// Author represents the intellectual creator or writer of a comic.
// A single comic may have multiple authors associated with it.
type Author struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	NameAlt   []string   `json:"name_alt"`
	Bio       *string    `json:"bio"`
	ImageURL  *string    `json:"image_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"` // soft-delete tracker
}

// Artist represents the illustrator or visual creator of a comic.
// Authors and Artists are tracked separately as some creators only do one role.
type Artist struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	NameAlt   []string   `json:"name_alt"`
	Bio       *string    `json:"bio"`
	ImageURL  *string    `json:"image_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"` // soft-delete tracker
}

// # Search Params

// ContributorFilter holds the parameters for a paginated author/artist search.
type ContributorFilter struct {
	Query string // Trigram search against name and namealt
}

// # Field Identifiers

// Global field names for validation and dynamic query mapping in the reference domain.
const (
	FieldName     = "name"
	FieldNameAlt  = "name_alt"
	FieldBio      = "bio"
	FieldImageURL = "image_url"
	FieldSlug     = "slug"
	FieldCode     = "code"
)
