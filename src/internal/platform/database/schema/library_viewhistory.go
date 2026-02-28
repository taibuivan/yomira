package schema

// LibraryViewHistoryTable represents the 'library.viewhistory' table
type LibraryViewHistoryTable struct {
	Table    string
	ID       string
	UserID   string
	ComicID  string
	ViewedAt string
}

// LibraryViewHistory is the schema definition for library.viewhistory
var LibraryViewHistory = LibraryViewHistoryTable{
	Table:    "library.viewhistory",
	ID:       "id",
	UserID:   "userid",
	ComicID:  "comicid",
	ViewedAt: "viewedat",
}
