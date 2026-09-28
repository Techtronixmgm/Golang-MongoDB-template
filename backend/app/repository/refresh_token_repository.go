package repository

import (
	"context"
	"time"

	"basic-app/models"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	Touch(ctx context.Context, tokenID string) error
	Revoke(ctx context.Context, tokenID string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}
