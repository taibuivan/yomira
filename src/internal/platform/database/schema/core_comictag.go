package schema

// ComicTagTable represents the 'core.comictag' table
type ComicTagTable struct {
	Table   string
	ComicID string
	TagID   string
}

// ComicTag is the schema definition for core.comictag
var ComicTag = ComicTagTable{
	Table:   "core.comictag",
	ComicID: "comicid",
	TagID:   "tagid",
}
