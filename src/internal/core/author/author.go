package author

import "time"

// Author represents the intellectual creator or writer of a comic.
type Author struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	NameAlt   []string   `json:"name_alt"`
	Bio       *string    `json:"bio"`
	ImageURL  *string    `json:"image_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"` // soft-delete tracker
}

// Filter holds the parameters for a paginated author search.
type Filter struct {
	Query string // Trigram search against name and namealt
}

// Global field names for validation
const (
	FieldName     = "name"
	FieldNameAlt  = "name_alt"
	FieldBio      = "bio"
	FieldImageURL = "image_url"
)
