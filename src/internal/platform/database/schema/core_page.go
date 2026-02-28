package schema

// CorePageTable represents the 'core.page' table
type CorePageTable struct {
	Table      string
	ID         string
	ChapterID  string
	PageNumber string
	ImageURL   string
}

// CorePage is the schema definition for core.page
var CorePage = CorePageTable{
	Table:      "core.page",
	ID:         "id",
	ChapterID:  "chapterid",
	PageNumber: "pagenumber",
	ImageURL:   "imageurl",
}

func (t CorePageTable) Columns() []string {
	return []string{t.ID, t.ChapterID, t.PageNumber, t.ImageURL}
}
