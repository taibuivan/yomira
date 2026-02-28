package schema

// LibraryCustomListItemTable represents the 'library.customlistitem' table
type LibraryCustomListItemTable struct {
	Table     string
	ListID    string
	ComicID   string
	SortOrder string
	AddedAt   string
}

// LibraryCustomListItem is the schema definition for library.customlistitem
var LibraryCustomListItem = LibraryCustomListItemTable{
	Table:     "library.customlistitem",
	ListID:    "listid",
	ComicID:   "comicid",
	SortOrder: "sortorder",
	AddedAt:   "addedat",
}
