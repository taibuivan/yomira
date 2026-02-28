package artist

import "context"

type Repository interface {
	ListArtists(context context.Context, f Filter, limit, offset int) ([]*Artist, int, error)
	GetArtist(context context.Context, id int) (*Artist, error)
	CreateArtist(context context.Context, a *Artist) error
	UpdateArtist(context context.Context, a *Artist) error
	DeleteArtist(context context.Context, id int) error
}
