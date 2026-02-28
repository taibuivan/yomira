package schema

// LibraryEntryTable represents the 'library.entry' table
type LibraryEntryTable struct {
	Table             string
	ID                string
	UserID            string
	ComicID           string
	ReadingStatus     string
	Score             string
	HasNew            string
	LastReadChapterID string
	LastReadAt        string
	CreatedAt         string
	UpdatedAt         string
}

// LibraryEntry is the schema definition for library.entry
var LibraryEntry = LibraryEntryTable{
	Table:             "library.entry",
	ID:                "id",
	UserID:            "userid",
	ComicID:           "comicid",
	ReadingStatus:     "readingstatus",
	Score:             "score",
	HasNew:            "hasnew",
	LastReadChapterID: "lastreadchapterid",
	LastReadAt:        "lastreadat",
	CreatedAt:         "createdat",
	UpdatedAt:         "updatedat",
}

func (t LibraryEntryTable) Columns() []string {
	return []string{
		t.ID, t.UserID, t.ComicID, t.ReadingStatus, t.Score, t.HasNew, t.LastReadChapterID,
		t.LastReadAt, t.CreatedAt, t.UpdatedAt,
	}
}
