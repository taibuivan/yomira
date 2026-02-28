package schema

// LibraryCustomListTable represents the 'library.customlist' table
type LibraryCustomListTable struct {
	Table      string
	ID         string
	UserID     string
	Name       string
	Visibility string
	CreatedAt  string
	UpdatedAt  string
	DeletedAt  string
}

// LibraryCustomList is the schema definition for library.customlist
var LibraryCustomList = LibraryCustomListTable{
	Table:      "library.customlist",
	ID:         "id",
	UserID:     "userid",
	Name:       "name",
	Visibility: "visibility",
	CreatedAt:  "createdat",
	UpdatedAt:  "updatedat",
	DeletedAt:  "deletedat",
}
