package services

import (
	"basic-app/auth"
	"basic-app/config"
	"basic-app/dto"
	"basic-app/models"
	"basic-app/repository"

	"basic-app/validation"
	"context"
	"crypto/hmac"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepository  repository.UserRepository
	settingsService *SettingsService
	config          config.Config
}

func NewAuthService(
	userRepository repository.UserRepository,
	settingsService *SettingsService,
	cfg config.Config,
) *AuthService {
	return &AuthService{
		userRepository:  userRepository,
		settingsService: settingsService,
		config:          cfg,
	}
}

// RegisterCustomer creates a new customer account.

func (s *AuthService) RegisterUser(
	ctx context.Context,
	req *dto.RegisterRequest,
) (*models.User, error) {

	registrationEnabled, err := s.settingsService.IsRegistrationEnabled(ctx)
	if err != nil {
		return nil, err
	}

	if !registrationEnabled {
		return nil, ErrRegistrationDisabled
	}

	firstName, err := validation.ValidateName(
		req.FirstName,
		"first name",
	)
	if err != nil {
		return nil, err
	}

	lastName, err := validation.ValidateName(
		req.LastName,
		"last name",
	)
	if err != nil {
		return nil, err
	}

	username, err := validation.ValidateUsername(req.Username)
	if err != nil {
		return nil, err
	}

	email, err := validation.ValidateEmail(req.Email, true)
	if err != nil {
		return nil, err
	}

	altEmail, err := validation.ValidateEmail(req.AltEmail, false)
	if err != nil {
		return nil, err
	}

	if altEmail != "" && altEmail == email {
		return nil, ErrAltEmailSameAsEmail
	}

	phone, err := validation.ValidatePhone(req.Phone)
	if err != nil {
		return nil, err
	}

	if err := validation.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Checks primary email against both email and alternate-email fields.
	existingUser, err := s.userRepository.FindByAnyEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// Checks optional alternate email against both email fields.
	if altEmail != "" {
		existingUser, err = s.userRepository.FindByAnyEmail(
			ctx,
			altEmail,
		)
		if err == nil && existingUser != nil {
			return nil, ErrAltEmailAlreadyExists
		}

		if err != nil && !errors.Is(
			err,
			ErrUserNotFound,
		) {
			return nil, err
		}
	}

	existingUser, err = s.userRepository.FindByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return nil, ErrUsernameAlreadyExists
	}

	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	existingUser, err = s.userRepository.FindByPhone(ctx, phone)
	if err == nil && existingUser != nil {
		return nil, ErrPhoneAlreadyExists
	}

	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := models.NewCustomerUser()

	user.FirstName = firstName
	user.LastName = lastName
	user.Username = username
	user.Email = email
	user.AltEmail = altEmail
	user.Phone = phone
	user.PasswordHash = string(passwordHash)

	if err := s.userRepository.Create(ctx, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ChangePassword changes a user's password.
func (s *AuthService) ChangePassword(
	ctx context.Context,
	userID string,
	req *dto.ChangePasswordRequest,
) error {
	if req == nil {
		return errors.New("change password request is required")
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidUserID
	}

	// Do not trim passwords.
	currentPassword := req.CurrentPassword
	newPassword := req.NewPassword

	if err := validation.ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify the user's current password.
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(currentPassword),
	); err != nil {
		return ErrInvalidPassword
	}

	// Hash the new password.
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	return s.userRepository.UpdatePassword(
		ctx,
		userID,
		string(passwordHash),
	)
}

// UpdateRefreshToken stores the hashed refresh token.
func (s *AuthService) UpdateRefreshToken(
	ctx context.Context,
	userID string,
	refreshTokenHash string,
) error {
	return s.userRepository.UpdateRefreshToken(
		ctx,
		userID,
		refreshTokenHash,
	)
}

// ClearRefreshToken removes the current refresh token.
func (s *AuthService) ClearRefreshToken(
	ctx context.Context,
	userID string,
) error {
	return s.userRepository.UpdateRefreshToken(
		ctx,
		userID,
		"",
	)
}

func (s *AuthService) Login(
	ctx context.Context,
	req *dto.LoginRequest,
) (*dto.LoginResult, error) {
	identifier := strings.ToLower(
		strings.TrimSpace(req.Identifier),
	)

	if identifier == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepository.FindByLoginIdentifier(
		ctx,
		identifier,
	)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if !user.Status {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := auth.GenerateToken(
		user.ID.Hex(),
		string(user.Role),
		s.config.JWTSecret,
		s.config.JWTExpiryHours,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExpiry, err := auth.GenerateRefreshToken(
		user.ID.Hex(),
		s.config.JWTSecret,
		s.config.RefreshTokenExpiryDays,
	)
	if err != nil {
		return nil, err
	}

	refreshTokenHash := auth.HashRefreshToken(refreshToken)

	if err := s.UpdateRefreshToken(
		ctx,
		user.ID.Hex(),
		refreshTokenHash,
	); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user.LastLoginAt = &now
	user.RefreshTokenHash = ""

	return &dto.LoginResult{
		User:          user,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		RefreshExpiry: refreshExpiry,
	}, nil
}

func (s *AuthService) RefreshToken(
	ctx context.Context,
	refreshToken string,
) (*dto.RefreshResult, error) {
	claims, err := auth.ValidateRefreshToken(
		refreshToken,
		s.config.JWTSecret,
	)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepository.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidRefreshToken
		}

		return nil, err
	}

	if !user.Status {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := auth.HashRefreshToken(refreshToken)

	if user.RefreshTokenHash == "" ||
		!hmac.Equal(
			[]byte(tokenHash),
			[]byte(user.RefreshTokenHash),
		) {
		return nil, ErrInvalidRefreshToken
	}

	accessToken, err := auth.GenerateToken(
		user.ID.Hex(),
		string(user.Role),
		s.config.JWTSecret,
		s.config.JWTExpiryHours,
	)
	if err != nil {
		return nil, err
	}

	newRefreshToken, refreshExpiry, err := auth.GenerateRefreshToken(
		user.ID.Hex(),
		s.config.JWTSecret,
		s.config.RefreshTokenExpiryDays,
	)
	if err != nil {
		return nil, err
	}

	newRefreshTokenHash := auth.HashRefreshToken(newRefreshToken)

	if err := s.UpdateRefreshToken(
		ctx,
		user.ID.Hex(),
		newRefreshTokenHash,
	); err != nil {
		return nil, err
	}

	return &dto.RefreshResult{
		AccessToken:   accessToken,
		RefreshToken:  newRefreshToken,
		RefreshExpiry: refreshExpiry,
	}, nil
}

func (s *AuthService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	claims, err := auth.ValidateRefreshToken(
		refreshToken,
		s.config.JWTSecret,
	)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	user, err := s.userRepository.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrInvalidRefreshToken
		}

		return err
	}

	tokenHash := auth.HashRefreshToken(refreshToken)

	if user.RefreshTokenHash == "" ||
		!hmac.Equal(
			[]byte(tokenHash),
			[]byte(user.RefreshTokenHash),
		) {
		return ErrInvalidRefreshToken
	}

	return s.ClearRefreshToken(ctx, user.ID.Hex())
}
