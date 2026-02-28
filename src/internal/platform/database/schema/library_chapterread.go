package schema

// CoreUserReadTable represents the 'library.chapterread' table (historically mapped as userread)
type CoreUserReadTable struct {
	Table     string
	UserID    string
	ChapterID string
	ReadAt    string
}

// CoreUserRead is the schema definition for library.chapterread
var CoreUserRead = CoreUserReadTable{
	Table:     "library.chapterread",
	UserID:    "userid",
	ChapterID: "chapterid",
	ReadAt:    "readat",
}

func (t CoreUserReadTable) Columns() []string {
	return []string{t.UserID, t.ChapterID, t.ReadAt}
}
