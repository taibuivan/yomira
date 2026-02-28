package schema

// RefLanguageTable represents the 'core.language' table
type RefLanguageTable struct {
	Table      string
	ID         string
	Code       string
	Name       string
	NativeName string
}

// RefLanguage is the schema definition for core.language
var RefLanguage = RefLanguageTable{
	Table:      "core.language",
	ID:         "id",
	Code:       "code",
	Name:       "name",
	NativeName: "nativename",
}

func (t RefLanguageTable) Columns() []string { return []string{t.ID, t.Code, t.Name, t.NativeName} }
