package schema

// RefArtistTable represents the 'core.artist' table
type RefArtistTable struct {
	Table     string
	ID        string
	Name      string
	NameAlt   string
	Bio       string
	ImageURL  string
	CreatedAt string
	UpdatedAt string
	DeletedAt string
}

// RefArtist is the schema definition for core.artist
var RefArtist = RefArtistTable{
	Table:     "core.artist",
	ID:        "id",
	Name:      "name",
	NameAlt:   "namealt",
	Bio:       "bio",
	ImageURL:  "imageurl",
	CreatedAt: "createdat",
	UpdatedAt: "updatedat",
	DeletedAt: "deletedat",
}

func (t RefArtistTable) Columns() []string {
	return []string{t.ID, t.Name, t.NameAlt, t.Bio, t.ImageURL, t.CreatedAt, t.UpdatedAt, t.DeletedAt}
}
