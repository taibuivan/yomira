package schema

// UserPreferencesTable represents the 'users.readingpreference' table
type UserPreferencesTable struct {
	Table         string
	UserID        string
	ReadingMode   string
	PageFit       string
	PreloadPages  string
	HideNSFW      string
	HideLanguages string
	DataSaver     string
}

// UserPreferences is the schema definition for users.readingpreference
var UserPreferences = UserPreferencesTable{
	Table:         "users.readingpreference",
	UserID:        "userid",
	ReadingMode:   "readingmode",
	PageFit:       "pagefit",
	PreloadPages:  "preloadpages",
	HideNSFW:      "hidensfw",
	HideLanguages: "hidelanguages",
	DataSaver:     "datasaver",
}

// Columns returns all standard column names
func (t UserPreferencesTable) Columns() []string {
	return []string{t.UserID, t.ReadingMode, t.PageFit, t.PreloadPages, t.HideNSFW, t.HideLanguages, t.DataSaver}
}
