package tag

import "context"

type Repository interface {
	ListTags(context context.Context) ([]*TagGroup, error)
	GetTagByID(context context.Context, id int) (*Tag, error)
	GetTagBySlug(context context.Context, slug string) (*Tag, error)
}
