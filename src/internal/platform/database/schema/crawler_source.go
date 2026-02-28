package schema

// CrawlerSourceTable represents the 'crawler.source' table
type CrawlerSourceTable struct {
	Table            string
	ID               string
	Name             string
	Slug             string
	BaseURL          string
	ExtensionID      string
	Config           string
	IsEnabled        string
	ConsecutiveFails string
	CreatedAt        string
	UpdatedAt        string
}

var CrawlerSource = CrawlerSourceTable{
	Table:            "crawler.source",
	ID:               "id",
	Name:             "name",
	Slug:             "slug",
	BaseURL:          "baseurl",
	ExtensionID:      "extensionid",
	Config:           "config",
	IsEnabled:        "isenabled",
	ConsecutiveFails: "consecutivefails",
	CreatedAt:        "createdat",
	UpdatedAt:        "updatedat",
}
