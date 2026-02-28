package language

import "context"

// Repository defines the data access contract.
type Repository interface {
	ListLanguages(context context.Context) ([]*Language, error)
	GetLanguageByCode(context context.Context, code string) (*Language, error)
}
