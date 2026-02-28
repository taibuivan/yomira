package schema

// UserAccountTable represents the 'users.account' table
type UserAccountTable struct {
	Table       string
	ID          string
	Username    string
	Email       string
	Password    string
	Role        string
	IsVerified  string
	IsActive    string
	LastLoginAt string
	DisplayName string
	AvatarURL   string
	Bio         string
	Website     string
	CreatedAt   string
	UpdatedAt   string
	DeletedAt   string
}

// UserAccount is the schema definition for users.account
var UserAccount = UserAccountTable{
	Table:       "users.account",
	ID:          "id",
	Username:    "username",
	Email:       "email",
	Password:    "passwordhash",
	Role:        "role",
	IsVerified:  "isverified",
	IsActive:    "isactive",
	LastLoginAt: "lastloginat",
	DisplayName: "displayname",
	AvatarURL:   "avatarurl",
	Bio:         "bio",
	Website:     "website",
	CreatedAt:   "createdat",
	UpdatedAt:   "updatedat",
	DeletedAt:   "deletedat",
}

// Columns returns all standard column names
func (t UserAccountTable) Columns() []string {
	return []string{
		t.ID, t.Username, t.Email, t.Password, t.Role, t.IsVerified,
		t.IsActive, t.LastLoginAt, t.DisplayName, t.AvatarURL, t.Bio,
		t.Website, t.CreatedAt, t.UpdatedAt, t.DeletedAt,
	}
}
