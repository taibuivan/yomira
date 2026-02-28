package schema

// CoreComicTable represents the 'core.comic' table
type CoreComicTable struct {
	Table           string
	ID              string
	Title           string
	AltTitle        string
	Slug            string
	Year            string
	Status          string
	ContentRating   string
	Demographic     string
	DefaultReadMode string
	OriginLanguage  string
	ViewCount       string
	FollowCount     string
	RatingAvg       string
	RatingBayesian  string
	RatingCount     string
	CoverURL        string
	Description     string
	IsLocked        string
	CreatedAt       string
	UpdatedAt       string
	DeletedAt       string
	SearchVector    string
	Links           string
}

// CoreComic is the schema definition for core.comic
var CoreComic = CoreComicTable{
	Table:           "core.comic",
	ID:              "id",
	Title:           "title",
	AltTitle:        "titlealt",
	Slug:            "slug",
	Year:            "year",
	Status:          "status",
	ContentRating:   "contentrating",
	Demographic:     "demographic",
	DefaultReadMode: "defaultreadmode",
	OriginLanguage:  "originlanguage",
	ViewCount:       "viewcount",
	FollowCount:     "followcount",
	RatingAvg:       "ratingavg",
	RatingBayesian:  "ratingbayesian",
	RatingCount:     "ratingcount",
	CoverURL:        "coverurl",
	Description:     "description",
	IsLocked:        "islocked",
	CreatedAt:       "createdat",
	UpdatedAt:       "updatedat",
	DeletedAt:       "deletedat",
	SearchVector:    "searchvector",
	Links:           "links",
}

func (t CoreComicTable) Columns() []string {
	return []string{
		t.ID, t.Title, t.AltTitle, t.Slug, t.Year, t.Status, t.ContentRating,
		t.Demographic, t.DefaultReadMode, t.OriginLanguage, t.ViewCount,
		t.FollowCount, t.RatingAvg, t.RatingBayesian, t.RatingCount,
		t.CoverURL, t.Description, t.IsLocked, t.CreatedAt, t.UpdatedAt, t.DeletedAt, t.SearchVector, t.Links,
	}
}
