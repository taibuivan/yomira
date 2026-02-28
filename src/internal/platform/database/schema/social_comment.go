package schema

// SocialCommentTable represents the 'social.comment' table
type SocialCommentTable struct {
	Table      string
	ID         string
	UserID     string
	ComicID    string
	ChapterID  string
	ParentID   string
	Body       string
	IsDeleted  string
	IsApproved string
	Upvotes    string
	Downvotes  string
	CreatedAt  string
	UpdatedAt  string
}

// SocialComment is the schema definition for social.comment
var SocialComment = SocialCommentTable{
	Table:      "social.comment",
	ID:         "id",
	UserID:     "userid",
	ComicID:    "comicid",
	ChapterID:  "chapterid",
	ParentID:   "parentid",
	Body:       "body",
	IsDeleted:  "isdeleted",
	IsApproved: "isapproved",
	Upvotes:    "upvotes",
	Downvotes:  "downvotes",
	CreatedAt:  "createdat",
	UpdatedAt:  "updatedat",
}
