package schema

// SystemAuditLogTable represents the 'system.auditlog' table
type SystemAuditLogTable struct {
	Table      string
	ID         string
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Before     string
	After      string
	IPAddress  string
	CreatedAt  string
}

var SystemAuditLog = SystemAuditLogTable{
	Table:      "system.auditlog",
	ID:         "id",
	ActorID:    "actorid",
	Action:     "action",
	EntityType: "entitytype",
	EntityID:   "entityid",
	Before:     "before",
	After:      "after",
	IPAddress:  "ipaddress",
	CreatedAt:  "createdat",
}
