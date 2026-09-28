package repository

import (
	"context"

	"basic-app/models"
)

type PageListFilter struct {
	Visibility *models.PageVisibility
	AuthorID   string
	Skip       int64
	Limit      int64
}

type PageRepository interface {
	Create(ctx context.Context, page *models.Page) error

	FindByID(ctx context.Context, id string) (*models.Page, error)
	FindBySlug(ctx context.Context, slug string) (*models.Page, error)

	List(
		ctx context.Context,
		filter PageListFilter,
	) ([]*models.Page, int64, error)

	Update(ctx context.Context, id string, page *models.Page) error
	Delete(ctx context.Context, id string) error
}
