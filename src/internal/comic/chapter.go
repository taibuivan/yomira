// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package comic

import "time"

// ── Aggregate ────────────────────────────────────────────────────────────────

// Chapter represents a single chapter (episode) of a comic.
//
// # Metadata
//
// It supports half-chapters (e.g., 12.5) via the Number field and tracks
// translation/publishing status.
type Chapter struct {
	ID          string
	ComicID     string
	Number      float64    // Supports half-chapters and specials.
	Title       string     // Optional; may be empty for untitled chapters.
	Language    string     // BCP-47 (e.g., "en", "vi").
	Translators []string   // Translator group names.
	ExternalURL string     // Source URL for crawler attribution.
	IsLocked    bool       // Indicates if the content requires a subscription or purchase.
	PublishedAt *time.Time // nil = draft.
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Page represents a single image page within a [Chapter].
//
// # Webtoons
//
// For vertical webtoons, all pages belong to the same "virtual" strip.
// Images are typically served via Cloudflare R2.
type Page struct {
	ID         string
	ChapterID  string
	PageNumber int
	ImageURL   string // CDN URL.
	Width      int
	Height     int
}

// ChapterFilter holds parameters for filtering a comic's chapter list.
//
// # Parameters
//   - Language: BCP-47 filter (e.g., "en").
//   - SortDir: Direction of sorting ("asc" or "desc") by chapter number.
type ChapterFilter struct {
	Language string
	SortDir  string
}
