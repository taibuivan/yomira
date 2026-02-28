package schema

// SystemSettingTable represents the 'system.setting' table
type SystemSettingTable struct {
	Table       string
	ID          string
	Key         string
	Value       string
	Description string
	UpdatedAt   string
}

var SystemSetting = SystemSettingTable{
	Table:       "system.setting",
	ID:          "id",
	Key:         "key",
	Value:       "value",
	Description: "description",
	UpdatedAt:   "updatedat",
}
