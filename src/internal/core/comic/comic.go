// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package comic defines the core domain entities for the Yomira catalogue.

It manages the lifecycle of serialised publications (Manga, Manhwa, Webtoons)
including metadata, chapter organization, and reading metrics.

Core Responsibility:

  - Catalogue: Defines statuses (Ongoing, Completed) and demographics (Shounen, Seinen).
  - Discovery: Manages tags (Genres/Themes) and titles (Alternative names/Translations).
  - Analytics: Tracks metrics like view counts and followers for ranking.

This package acts as the source of truth for all content-related data models.
*/
package comic

import "time"

// # Domain Enums

// Status represents the publication status of a comic.
type Status string

const (
	// StatusOngoing indicates the publication is actively updating.
	StatusOngoing Status = "ongoing"

	// StatusCompleted indicates no further chapters are expected.
	StatusCompleted Status = "completed"

	// StatusHiatus indicates the publication is paused indefinitely.
	StatusHiatus Status = "hiatus"

	// StatusCancelled indicates the publication has been permanently discontinued.
	StatusCancelled Status = "cancelled"

	// StatusUnknown is the default when status information is unavailable.
	StatusUnknown Status = "unknown"
)

// IsValid reports whether s is a recognised [Status] value.
func (s Status) IsValid() bool {
	switch s {
	case
		StatusOngoing,
		StatusCompleted,
		StatusHiatus,
		StatusCancelled,
		StatusUnknown:
		return true
	}
	return false
}

// ContentRating classifies the audience suitability of a comic.
type ContentRating string

const (
	// ContentRatingSafe is suitable for all audiences.
	ContentRatingSafe ContentRating = "safe"

	// ContentRatingSuggestive contains mildly suggestive content.
	ContentRatingSuggestive ContentRating = "suggestive"

	// ContentRatingExplicit is intended for adult audiences.
	ContentRatingExplicit ContentRating = "explicit"
)

// Demographic classifies the target readership of a comic (e.g. Shounen, Seinen).
type Demographic string

const (
	DemographicShounen Demographic = "shounen"
	DemographicShoujo  Demographic = "shoujo"
	DemographicSeinen  Demographic = "seinen"
	DemographicJosei   Demographic = "josei"
	DemographicUnknown Demographic = "unknown"
)

// ReadMode describes the default reading direction (e.g. Manga RTL vs Webtoon Vertical).
type ReadMode string

const (
	ReadModeRTL      ReadMode = "rtl"
	ReadModeLTR      ReadMode = "ltr"
	ReadModeVertical ReadMode = "vertical"
	ReadModeWebtoon  ReadMode = "webtoon"
)

// # Core Entities

// Comic is the central aggregate of the Yomira domain.
// It represents a single serialised publication in the catalogue.
type Comic struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	TitleAlt        []string          `json:"title_alt"` // Alternative/romanised titles
	Slug            string            `json:"slug"`      // URL-safe identifier
	Synopsis        string            `json:"synopsis"`
	CoverURL        string            `json:"cover_url"`
	Status          Status            `json:"status"`
	ContentRating   ContentRating     `json:"content_rating"`
	Demographic     Demographic       `json:"demographic"`
	DefaultReadMode ReadMode          `json:"default_read_mode"`
	OriginLanguage  string            `json:"origin_language"` // BCP-47 language tag (e.g. "ja", "ko", "zh")
	Year            *int16            `json:"year,omitempty"`  // Discovery year
	Links           map[string]string `json:"links,omitempty"` // External links keyed by provider (e.g. "mal", "al")
	Tags            []Tag             `json:"tags,omitempty"`  // Genre/theme associations

	// # Junction IDs (Input only)
	AuthorIDs []int `json:"author_ids,omitempty"`
	ArtistIDs []int `json:"artist_ids,omitempty"`
	TagIDs    []int `json:"tag_ids,omitempty"`

	// # Computed Metrics
	// These fields are updated asynchronously by background analytics workers.
	ViewCount      int64   `json:"view_count"`
	FollowCount    int64   `json:"follow_count"`
	RatingAvg      float64 `json:"rating_avg"`
	RatingBayesian float64 `json:"rating_bayesian"`
	RatingCount    int     `json:"rating_count"`

	IsLocked  bool       `json:"is_locked"` // True while under moderation review
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"` // nil = active; non-nil = soft-deleted
}

// Tag represents a genre, theme, or content classifier attached to a [Comic].
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Title represents a specific title for a comic in a given language.
type Title struct {
	ComicID  string `json:"comic_id"`
	Language string `json:"language"`
	Title    string `json:"title"`
}

// Relation defines a link between two comics (e.g. Sequel, Prequel).
type Relation struct {
	FromComicID string `json:"from_comic_id"`
	ToComicID   string `json:"to_comic_id"`
	Type        string `json:"type"`     // sequel, prequel, spinoff, adaptation, etc
	ToTitle     string `json:"to_title"` // Denormalized for display
}

// Cover represents a specific volume or seasonal cover for a comic.
type Cover struct {
	ID          string    `json:"id"`
	ComicID     string    `json:"comic_id"`
	Volume      *int      `json:"volume,omitempty"`
	ImageURL    string    `json:"image_url"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Art represents a fanart or official gallery image associated with a comic.
type Art struct {
	ID         string    `json:"id"`
	ComicID    string    `json:"comic_id"`
	UploaderID string    `json:"uploader_id"`
	ImageURL   string    `json:"image_url"`
	IsApproved bool      `json:"is_approved"`
	CreatedAt  time.Time `json:"created_at"`
}

// # Search & Filtering

// Filter holds the parameters for a filtered comic list query.
type Filter struct {
	Status            []Status        `json:"status,omitempty"`
	ContentRating     []ContentRating `json:"content_rating,omitempty"`
	Demographic       []Demographic   `json:"demographic,omitempty"`
	OriginLanguage    []string        `json:"origin_language,omitempty"`
	IncludedTags      []int           `json:"included_tags,omitempty"`
	ExcludedTags      []int           `json:"excluded_tags,omitempty"`
	IncludedAuthors   []int           `json:"included_authors,omitempty"`
	IncludedArtists   []int           `json:"included_artists,omitempty"`
	AvailableLanguage string          `json:"available_language,omitempty"`
	Year              *int16          `json:"year,omitempty"`
	Query             string          `json:"q,omitempty"`        // Full-text search term
	Sort              string          `json:"sort,omitempty"`     // latest, popular, rating, etc
	SortDir           string          `json:"sort_dir,omitempty"` // "asc" or "desc"
}

// # Field Identifiers

// Global field names for validation and dynamic query mapping.
const (
	FieldID              = "id"
	FieldTitle           = "title"
	FieldTitleAlt        = "title_alt"
	FieldSlug            = "slug"
	FieldSynopsis        = "synopsis"
	FieldCoverURL        = "cover_url"
	FieldStatus          = "status"
	FieldContentRating   = "content_rating"
	FieldDemographic     = "demographic"
	FieldDefaultReadMode = "default_read_mode"
	FieldOriginLanguage  = "origin_language"
	FieldYear            = "year"
	FieldLinks           = "links"
	FieldTagIDs          = "tag_ids"
	FieldAuthorIDs       = "author_ids"
	FieldArtistIDs       = "artist_ids"
)

// Field identifiers for the [Chapter] domain.
const (
	FieldChapterID     = "id"
	FieldComicID       = "comic_id"
	FieldChapterNumber = "chapter_number"
	FieldChapterTitle  = "title"
	FieldLanguage      = "language"
	FieldImageURL      = "image_url"
	FieldUploaderID    = "uploader_id"
	FieldItems         = "items"
	FieldTotal         = "total"
	FieldMessage       = "message"
)
