package schema

// RefAuthorTable represents the 'core.author' table
type RefAuthorTable struct {
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

// RefAuthor is the schema definition for core.author
var RefAuthor = RefAuthorTable{
	Table:     "core.author",
	ID:        "id",
	Name:      "name",
	NameAlt:   "namealt",
	Bio:       "bio",
	ImageURL:  "imageurl",
	CreatedAt: "createdat",
	UpdatedAt: "updatedat",
	DeletedAt: "deletedat",
}

func (t RefAuthorTable) Columns() []string {
	return []string{t.ID, t.Name, t.NameAlt, t.Bio, t.ImageURL, t.CreatedAt, t.UpdatedAt, t.DeletedAt}
}
