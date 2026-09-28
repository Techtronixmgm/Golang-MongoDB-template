package services

import (
	"context"

	"basic-app/models"
	"basic-app/repository"
)

type SettingsService struct {
	settingsRepository repository.ApplicationSettingsRepository
}

func NewSettingsService(
	settingsRepository repository.ApplicationSettingsRepository,
) *SettingsService {
	return &SettingsService{
		settingsRepository: settingsRepository,
	}
}

func (s *SettingsService) GetSettings(
	ctx context.Context,
) (*models.ApplicationSettings, error) {
	return s.settingsRepository.Get(ctx)
}

func (s *SettingsService) IsRegistrationEnabled(
	ctx context.Context,
) (bool, error) {
	settings, err := s.settingsRepository.Get(ctx)
	if err != nil {
		return false, err
	}

	return settings.RegistrationEnabled, nil
}

func (s *SettingsService) UpdateRegistrationEnabled(
	ctx context.Context,
	enabled bool,
	updatedBy string,
) error {
	return s.settingsRepository.UpdateRegistrationEnabled(
		ctx,
		enabled,
		updatedBy,
	)
}
