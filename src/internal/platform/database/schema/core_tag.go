package schema

// RefTagTable represents the 'core.tag' table
type RefTagTable struct {
	Table       string
	ID          string
	GroupID     string
	Name        string
	Slug        string
	Description string
}

// RefTag is the schema definition for core.tag
var RefTag = RefTagTable{
	Table:       "core.tag",
	ID:          "id",
	GroupID:     "groupid",
	Name:        "name",
	Slug:        "slug",
	Description: "description",
}

func (t RefTagTable) Columns() []string {
	return []string{t.ID, t.GroupID, t.Name, t.Slug, t.Description}
}
