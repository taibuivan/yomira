package author

import "context"

type Repository interface {
	ListAuthors(context context.Context, f Filter, limit, offset int) ([]*Author, int, error)
	GetAuthor(context context.Context, id int) (*Author, error)
	CreateAuthor(context context.Context, a *Author) error
	UpdateAuthor(context context.Context, a *Author) error
	DeleteAuthor(context context.Context, id int) error
}
