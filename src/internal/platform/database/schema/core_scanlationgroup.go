package schema

// CoreGroupTable represents the 'core.scanlationgroup' table
type CoreGroupTable struct {
	Table               string
	ID                  string
	Name                string
	Slug                string
	Description         string
	Website             string
	Discord             string
	Twitter             string
	Patreon             string
	Youtube             string
	MangaUpdates        string
	IsOfficialPublisher string
	IsActive            string
	IsFocused           string
	FollowCount         string
	VerifiedAt          string
	CreatedAt           string
	UpdatedAt           string
	DeletedAt           string
}

// CoreGroup is the schema definition for core.scanlationgroup
var CoreGroup = CoreGroupTable{
	Table:               "core.scanlationgroup",
	ID:                  "id",
	Name:                "name",
	Slug:                "slug",
	Description:         "description",
	Website:             "website",
	Discord:             "discord",
	Twitter:             "twitter",
	Patreon:             "patreon",
	Youtube:             "youtube",
	MangaUpdates:        "mangaupdates",
	IsOfficialPublisher: "isofficialpublisher",
	IsActive:            "isactive",
	IsFocused:           "isfocused",
	FollowCount:         "followcount",
	VerifiedAt:          "verifiedat",
	CreatedAt:           "createdat",
	UpdatedAt:           "updatedat",
	DeletedAt:           "deletedat",
}

func (t CoreGroupTable) Columns() []string {
	return []string{
		t.ID, t.Name, t.Slug, t.Description, t.Website, t.Discord, t.Twitter, t.Patreon, t.Youtube,
		t.MangaUpdates, t.IsOfficialPublisher, t.IsActive, t.IsFocused, t.FollowCount, t.VerifiedAt,
		t.CreatedAt, t.UpdatedAt, t.DeletedAt,
	}
}
