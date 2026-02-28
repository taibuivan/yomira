package schema

// UserSessionTable represents the 'users.session' table
type UserSessionTable struct {
	Table      string
	ID         string
	UserID     string
	TokenHash  string
	DeviceName string
	IPAddress  string
	UserAgent  string
	IsRevoked  string
	ExpiresAt  string
	RevokedAt  string
	CreatedAt  string
}

// UserSession is the schema definition for users.session
var UserSession = UserSessionTable{
	Table:      "users.session",
	ID:         "id",
	UserID:     "userid",
	TokenHash:  "tokenhash",
	DeviceName: "devicename",
	IPAddress:  "ipaddress",
	UserAgent:  "useragent",
	IsRevoked:  "isrevoked",
	ExpiresAt:  "expiresat",
	RevokedAt:  "revokedat",
	CreatedAt:  "createdat",
}

// Columns returns all standard column names
func (t UserSessionTable) Columns() []string {
	return []string{
		t.ID, t.UserID, t.TokenHash, t.DeviceName, t.IPAddress, t.UserAgent, t.IsRevoked, t.ExpiresAt, t.RevokedAt, t.CreatedAt,
	}
}
