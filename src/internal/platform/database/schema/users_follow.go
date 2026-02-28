package schema

// UserFollowTable represents the 'users.follow' table
type UserFollowTable struct {
	Table       string
	FollowerID  string
	FollowingID string
	CreatedAt   string
}

// UserFollow is the schema definition for users.follow
var UserFollow = UserFollowTable{
	Table:       "users.follow",
	FollowerID:  "followerid",
	FollowingID: "followingid",
	CreatedAt:   "createdat",
}

// Columns returns all standard column names
func (t UserFollowTable) Columns() []string {
	return []string{t.FollowerID, t.FollowingID, t.CreatedAt}
}
