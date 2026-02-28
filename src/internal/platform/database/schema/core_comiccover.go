package schema

// CoreComicCoverTable represents the 'core.comiccover' table
type CoreComicCoverTable struct {
	Table       string
	ID          string
	ComicID     string
	Volume      string
	ImageURL    string
	Description string
	CreatedAt   string
}

// CoreComicCover is the schema definition for core.comiccover
var CoreComicCover = CoreComicCoverTable{
	Table:       "core.comiccover",
	ID:          "id",
	ComicID:     "comicid",
	Volume:      "volume",
	ImageURL:    "imageurl",
	Description: "description",
	CreatedAt:   "createdat",
}
