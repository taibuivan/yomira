// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "time"

// ComicStatus represents the publication status of a comic.
type ComicStatus string

const (
	// ComicStatusOngoing indicates the publication is actively updating.
	ComicStatusOngoing ComicStatus = "ongoing"
	// ComicStatusCompleted indicates no further chapters are expected.
	ComicStatusCompleted ComicStatus = "completed"
	// ComicStatusHiatus indicates the publication is paused indefinitely.
	ComicStatusHiatus ComicStatus = "hiatus"
	// ComicStatusCancelled indicates the publication has been permanently discontinued.
	ComicStatusCancelled ComicStatus = "cancelled"
	// ComicStatusUnknown is the default when status information is unavailable.
	ComicStatusUnknown ComicStatus = "unknown"
)

// IsValid reports whether s is a recognised [ComicStatus] value.
func (s ComicStatus) IsValid() bool {
	switch s {
	case ComicStatusOngoing, ComicStatusCompleted,
		ComicStatusHiatus, ComicStatusCancelled, ComicStatusUnknown:
		return true
	}
	return false
}

// ContentRating classifies the audience suitability of a comic.
type ContentRating string

const (
	// ContentRatingEveryone is suitable for all audiences.
	ContentRatingEveryone ContentRating = "everyone"
	// ContentRatingTeen is intended for readers 13 and older.
	ContentRatingTeen ContentRating = "teen"
	// ContentRatingSuggestive contains mildly suggestive content.
	ContentRatingSuggestive ContentRating = "suggestive"
	// ContentRatingMature is intended for adult audiences.
	ContentRatingMature ContentRating = "mature"
)

// Demographic classifies the target readership of a comic.
type Demographic string

const (
	// DemographicShounen targets young male readers.
	DemographicShounen Demographic = "shounen"
	// DemographicShoujo targets young female readers.
	DemographicShoujo Demographic = "shoujo"
	// DemographicSeinen targets adult male readers.
	DemographicSeinen Demographic = "seinen"
	// DemographicJosei targets adult female readers.
	DemographicJosei Demographic = "josei"
	// DemographicUnknown is the default when demographic is unclassified.
	DemographicUnknown Demographic = "unknown"
)

// ReadMode describes the default reading direction for a comic.
type ReadMode string

const (
	// ReadModeRTL is right-to-left (manga-style).
	ReadModeRTL ReadMode = "rtl"
	// ReadModeLTR is left-to-right (manhwa/manhua-style).
	ReadModeLTR ReadMode = "ltr"
	// ReadModeVertical is long-strip vertical scroll (webtoon-style).
	ReadModeVertical ReadMode = "vertical"
)

// Comic is the central aggregate of the Yomira domain.
//
// # Overview
//
// It represents a single serialised publication (manga, manhwa, manhua, webtoon, etc.)
// in the catalogue. It tracks both static metadata and computed metrics.
type Comic struct {
	ID              string
	Title           string
	TitleAlt        []string // Alternative/romanised titles.
	Slug            string   // URL-safe identifier (e.g. "solo-leveling").
	Synopsis        string
	CoverURL        string
	Status          ComicStatus
	ContentRating   ContentRating
	Demographic     Demographic
	DefaultReadMode ReadMode
	OriginLanguage  string            // BCP-47 language tag (e.g. "ja", "ko", "zh").
	Year            *int16            // First publication year; nil if unknown.
	Links           map[string]string // External links keyed by provider (e.g. "mal", "al").
	Tags            []Tag             // Genre/theme tags.

	// # Computed Metrics
	//
	// These fields are updated by background workers, not by direct writes
	// from the primary API handlers.
	ViewCount      int64
	FollowCount    int64
	RatingAvg      float64
	RatingBayesian float64
	RatingCount    int

	IsLocked  bool // True while under moderation review.
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // nil = active; non-nil = soft-deleted.
}

// Tag represents a genre, theme, or content tag attached to a [Comic].
type Tag struct {
	ID   string
	Name string
	Slug string
}

// ComicFilter holds the parameters for a filtered comic list query.
//
// # Sorting
//
// Available SortBy values: "viewcount", "rating", "followcount", "updatedat".
// SortDir can be "asc" or "desc".
type ComicFilter struct {
	Status        *ComicStatus
	ContentRating *ContentRating
	Demographic   *Demographic
	TagSlugs      []string
	Language      string // Filter by origin language.
	Year          *int16
	Query         string // Full-text search term.
	SortBy        string
	SortDir       string
}
