// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package comic provides the domain models for chapters and visual content.

It manages the hierarchical organization of images within chapters, handling
metadata such as chapter numbering, language localization, and translator attribution.

# Core Responsibility

  - Serialisation: Manages [Chapter] ordering (including half-chapters like 1.5).
  - Content Delivery: Defines the [Page] structure for CDN-distributed images.
  - Localization: Tracks [Language] specific releases for multi-lingual catalogues.

This file complements the core [Comic] aggregate by defining its structural parts.
*/
package comic

import "time"

// # Chapter Aggregate

// Chapter represents a single chapter (episode) of a comic.
// It acts as the container for a sequence of image pages.
type Chapter struct {
	ID          string
	ComicID     string
	Number      float64    // Supports half-chapters (e.g. 12.5) and specials
	Title       string     // Optional; may be empty for untitled chapters
	Language    string     // BCP-47 identifier (e.g. "en", "vi")
	Translators []string   // Scanslation group names or individual credits
	ExternalURL string     // Link for crawler attribution or official source
	IsLocked    bool       // True if content is behind a paywall or premium tier
	PublishedAt *time.Time // nil indicates draft or scheduled status
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // soft-delete tracker
}

// # Image Delivery

// Page represents a single image page within a [Chapter].
// It contains metadata required for optimized frontend rendering and CDN fetching.
type Page struct {
	ID         string
	ChapterID  string
	PageNumber int
	ImageURL   string // Content Delivery Network (CDN) URL
	Width      int    // Pixel width for pre-rendering layout
	Height     int    // Pixel height for pre-rendering layout
}

// # Filter Criteria

// ChapterFilter holds parameters for filtering a comic's chapter list.
type ChapterFilter struct {
	Language string // BCP-47 filter (e.g. "en", "ja")
	SortDir  string // Direction of sorting ("asc" or "desc") by chapter number
}
