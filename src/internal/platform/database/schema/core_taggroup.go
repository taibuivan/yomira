package schema

// RefTagGroupTable represents the 'core.taggroup' table
type RefTagGroupTable struct {
	Table     string
	ID        string
	Name      string
	Slug      string
	SortOrder string
}

// RefTagGroup is the schema definition for core.taggroup
var RefTagGroup = RefTagGroupTable{
	Table:     "core.taggroup",
	ID:        "id",
	Name:      "name",
	Slug:      "slug",
	SortOrder: "sortorder",
}

func (t RefTagGroupTable) Columns() []string { return []string{t.ID, t.Name, t.Slug, t.SortOrder} }
