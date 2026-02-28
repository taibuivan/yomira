package schema

// SocialComicRatingTable represents the 'social.comicrating' table
type SocialComicRatingTable struct {
	Table     string
	ID        string
	UserID    string
	ComicID   string
	Score     string
	CreatedAt string
	UpdatedAt string
}

// SocialComicRating is the schema definition for social.comicrating
var SocialComicRating = SocialComicRatingTable{
	Table:     "social.comicrating",
	ID:        "id",
	UserID:    "userid",
	ComicID:   "comicid",
	Score:     "score",
	CreatedAt: "createdat",
	UpdatedAt: "updatedat",
}
