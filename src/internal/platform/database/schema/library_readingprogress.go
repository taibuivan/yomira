package schema

// LibraryReadingProgressTable represents the 'library.readingprogress' table
type LibraryReadingProgressTable struct {
	Table      string
	ID         string
	UserID     string
	ComicID    string
	ChapterID  string
	PageNumber string
	UpdatedAt  string
}

// LibraryReadingProgress is the schema definition for library.readingprogress
var LibraryReadingProgress = LibraryReadingProgressTable{
	Table:      "library.readingprogress",
	ID:         "id",
	UserID:     "userid",
	ComicID:    "comicid",
	ChapterID:  "chapterid",
	PageNumber: "pagenumber",
	UpdatedAt:  "updatedat",
}
