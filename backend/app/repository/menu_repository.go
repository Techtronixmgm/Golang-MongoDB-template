package repository

import (
	"basic-app/models"
	"context"
)

type MenuRepository interface {
	Create(ctx context.Context, menu *models.Menu) error
	FindByID(ctx context.Context, id string) (*models.Menu, error)
	FindByLocation(ctx context.Context, location models.MenuLocation) (*models.Menu, error)
	List(ctx context.Context) ([]*models.Menu, error)
	Update(ctx context.Context, id string, menu *models.Menu) error
	Delete(ctx context.Context, id string) error
}
