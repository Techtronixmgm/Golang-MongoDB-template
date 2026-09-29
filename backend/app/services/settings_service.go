package services

import (
	"context"

	"basic-app/config"
	"basic-app/models"
	"basic-app/repository"
)

type SettingsService struct {
	settingsRepository repository.ApplicationSettingsRepository
	config             config.Config
}

func NewSettingsService(
	settingsRepository repository.ApplicationSettingsRepository,
	cfg config.Config,
) *SettingsService {
	return &SettingsService{
		settingsRepository: settingsRepository,
		config:             cfg,
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

func (s *SettingsService) IsTwoFactorEnabled(
	ctx context.Context,
) (bool, error) {
	settings, err := s.settingsRepository.Get(ctx)
	if err != nil {
		return false, err
	}

	return settings.TwoFactorEnabled, nil
}

func (s *SettingsService) UpdateTwoFactorEnabled(
	ctx context.Context,
	enabled bool,
	updatedBy string,
) error {
	if enabled {
		if err := s.config.ValidateTOTPEncryptionKey(); err != nil {
			return err
		}
	}

	return s.settingsRepository.UpdateTwoFactorEnabled(
		ctx,
		enabled,
		updatedBy,
	)
}
