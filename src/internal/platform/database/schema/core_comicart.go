package schema

// CoreComicArtTable represents the 'core.comicart' table
type CoreComicArtTable struct {
	Table      string
	ID         string
	ComicID    string
	UploaderID string
	ImageURL   string
	IsApproved string
	CreatedAt  string
}

// CoreComicArt is the schema definition for core.comicart
var CoreComicArt = CoreComicArtTable{
	Table:      "core.comicart",
	ID:         "id",
	ComicID:    "comicid",
	UploaderID: "uploaderid",
	ImageURL:   "imageurl",
	IsApproved: "isapproved",
	CreatedAt:  "createdat",
}
