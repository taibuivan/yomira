package artist

import "time"

// Artist represents the illustrator or visual creator of a comic.
type Artist struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	NameAlt   []string   `json:"name_alt"`
	Bio       *string    `json:"bio"`
	ImageURL  *string    `json:"image_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"` // soft-delete tracker
}

// Filter holds the parameters for a paginated artist search.
type Filter struct {
	Query string // Trigram search against name and namealt
}

const (
	FieldName     = "name"
	FieldNameAlt  = "name_alt"
	FieldBio      = "bio"
	FieldImageURL = "image_url"
)
