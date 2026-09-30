package repository

import (
	"context"

	"basic-app/models"
)

type ApplicationSettingsRepository interface {
	Get(ctx context.Context) (*models.ApplicationSettings, error)
	EnsureDefaults(ctx context.Context) error
	UpdateRegistrationEnabled(ctx context.Context, enabled bool, updatedBy string) error
	UpdateTwoFactorEnabled(ctx context.Context, enabled bool, updatedBy string) error
	UpdateLoginIdentifiers(ctx context.Context, primaryEmail bool, username bool, phone bool, altEmail bool, updatedBy string) error
}
