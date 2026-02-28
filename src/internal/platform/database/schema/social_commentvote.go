package schema

// SocialCommentVoteTable represents the 'social.commentvote' table
type SocialCommentVoteTable struct {
	Table     string
	UserID    string
	CommentID string
	Vote      string
}

// SocialCommentVote is the schema definition for social.commentvote
var SocialCommentVote = SocialCommentVoteTable{
	Table:     "social.commentvote",
	UserID:    "userid",
	CommentID: "commentid",
	Vote:      "vote",
}
