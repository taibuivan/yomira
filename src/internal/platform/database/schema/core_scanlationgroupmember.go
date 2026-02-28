package schema

// CoreMemberTable represents the 'core.scanlationgroupmember' table
type CoreMemberTable struct {
	Table    string
	GroupID  string
	UserID   string
	Role     string
	JoinedAt string
}

// CoreMember is the schema definition for core.scanlationgroupmember
var CoreMember = CoreMemberTable{
	Table:    "core.scanlationgroupmember",
	GroupID:  "groupid",
	UserID:   "userid",
	Role:     "role",
	JoinedAt: "joinedat",
}

func (t CoreMemberTable) Columns() []string {
	return []string{t.GroupID, t.UserID, t.Role, t.JoinedAt}
}
