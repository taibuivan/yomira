package schema

// CoreComicTitleTable represents the 'core.comictitle' table
type CoreComicTitleTable struct {
	Table      string
	ComicID    string
	LanguageID string
	Title      string
}

// CoreComicTitle is the schema definition for core.comictitle
var CoreComicTitle = CoreComicTitleTable{
	Table:      "core.comictitle",
	ComicID:    "comicid",
	LanguageID: "languageid",
	Title:      "title",
}
