package schema

// CrawlerComicSourceTable represents the 'crawler.comicsource' table
type CrawlerComicSourceTable struct {
	Table       string
	ID          string
	ComicID     string
	SourceID    string
	SourceIDExt string
	SourceURL   string
	IsActive    string
	LastCrawlAt string
	CreatedAt   string
}

var CrawlerComicSource = CrawlerComicSourceTable{
	Table:       "crawler.comicsource",
	ID:          "id",
	ComicID:     "comicid",
	SourceID:    "sourceid",
	SourceIDExt: "sourceid_ext",
	SourceURL:   "sourceurl",
	IsActive:    "isactive",
	LastCrawlAt: "lastcrawlat",
	CreatedAt:   "createdat",
}
