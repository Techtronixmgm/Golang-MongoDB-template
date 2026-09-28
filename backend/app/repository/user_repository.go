package repository

import (
	"context"

	"basic-app/models"
)

type CustomerListFilter struct {
	Status *bool
	Search string
	Skip   int64
	Limit  int64
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error

	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByAnyEmail(ctx context.Context, email string) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	FindByLoginIdentifier(
		ctx context.Context,
		identifier string,
	) (*models.User, error)

	Update(ctx context.Context, user *models.User) error

	UpdateProfilePic(
		ctx context.Context,
		userID string,
		profilePic string,
	) error

	UpdatePassword(
		ctx context.Context,
		userID string,
		passwordHash string,
	) error

	UpdateRefreshToken(
		ctx context.Context,
		userID string,
		refreshTokenHash string,
	) error

	UserStatus(
		ctx context.Context, userID string, status bool) error

	ListCustomers(
		ctx context.Context,
		filter CustomerListFilter,
	) ([]models.User, int64, error)
}
