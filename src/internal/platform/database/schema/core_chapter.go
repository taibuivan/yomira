package schema

// CoreChapterTable represents the 'core.chapter' table
type CoreChapterTable struct {
	Table             string
	ID                string
	ComicID           string
	LanguageID        string
	ScanlationGroupID string
	UploaderID        string
	Number            string
	ChapterNumber     string
	Title             string
	Volume            string
	SyncState         string
	ExternalURL       string
	IsOfficial        string
	IsLocked          string
	ViewCount         string
	PublishedAt       string
	CreatedAt         string
	UpdatedAt         string
	DeletedAt         string
}

// CoreChapter is the schema definition for core.chapter
var CoreChapter = CoreChapterTable{
	Table:             "core.chapter",
	ID:                "id",
	ComicID:           "comicid",
	LanguageID:        "languageid",
	ScanlationGroupID: "scanlationgroupid",
	UploaderID:        "uploaderid",
	Number:            "chapternumber",
	ChapterNumber:     "chapternumber",
	Title:             "title",
	Volume:            "volume",
	SyncState:         "syncstate",
	ExternalURL:       "externalurl",
	IsOfficial:        "isofficial",
	IsLocked:          "islocked",
	ViewCount:         "viewcount",
	PublishedAt:       "publishedat",
	CreatedAt:         "createdat",
	UpdatedAt:         "updatedat",
	DeletedAt:         "deletedat",
}

func (t CoreChapterTable) Columns() []string {
	return []string{
		t.ID, t.ComicID, t.LanguageID, t.ScanlationGroupID, t.UploaderID, t.Number, t.ChapterNumber, t.Title,
		t.Volume, t.SyncState, t.ExternalURL, t.IsOfficial, t.IsLocked, t.ViewCount,
		t.PublishedAt, t.CreatedAt, t.UpdatedAt, t.DeletedAt,
	}
}
