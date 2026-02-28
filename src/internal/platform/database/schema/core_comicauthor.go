package schema

// ComicAuthorTable represents the 'core.comicauthor' table
type ComicAuthorTable struct {
	Table    string
	ComicID  string
	AuthorID string
}

// ComicAuthor is the schema definition for core.comicauthor
var ComicAuthor = ComicAuthorTable{
	Table:    "core.comicauthor",
	ComicID:  "comicid",
	AuthorID: "authorid",
}
