package schema

// CoreFollowTable represents the 'core.scanlationgroupfollow' table
type CoreFollowTable struct {
	Table     string
	GroupID   string
	UserID    string
	CreatedAt string
}

// CoreFollow is the schema definition for core.scanlationgroupfollow
var CoreFollow = CoreFollowTable{
	Table:     "core.scanlationgroupfollow",
	GroupID:   "groupid",
	UserID:    "userid",
	CreatedAt: "createdat",
}

func (t CoreFollowTable) Columns() []string {
	return []string{t.GroupID, t.UserID, t.CreatedAt}
}
