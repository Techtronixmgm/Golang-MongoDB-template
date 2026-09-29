package services

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"basic-app/dto"
	"basic-app/models"
	"basic-app/repository"

	"basic-app/validation"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepository        repository.UserRepository
	totpService           *TOTPService
	totpEncryptionService *TOTPEncryptionService
}

func NewUserService(
	userRepository repository.UserRepository,
	totpService *TOTPService,
	totpEncryptionService *TOTPEncryptionService,
) *UserService {
	return &UserService{
		userRepository:        userRepository,
		totpService:           totpService,
		totpEncryptionService: totpEncryptionService,
	}
}

func (s *UserService) GetMe(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	return s.userRepository.FindByID(ctx, userID)
}

// GetByID returns a user by ID.
func (s *UserService) GetByID(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	return s.userRepository.FindByID(
		ctx,
		userID,
	)
}

func (s *UserService) GetCustomerByID(
	ctx context.Context,
	customerID string,
) (*models.User, error) {
	user, err := s.userRepository.FindByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if user.Role != models.RoleCustomer {
		return nil, ErrInvalidUserRole
	}

	return user, nil
}

// UpdateProfile updates general user information.
func (s *UserService) UpdateProfile(
	ctx context.Context,
	userID string,
	req *dto.UpdateUserProfileRequest,
) (*models.User, error) {
	if req == nil {
		return nil, ErrNoFieldsToUpdate
	}

	currentUser, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !currentUser.Status {
		return nil, ErrInvalidCredentials
	}

	// Work on a copy. The database is changed only after all validation passes.
	user := *currentUser
	hasUpdates := false

	if req.FirstName != nil {
		firstName, err := validation.ValidateName(
			*req.FirstName,
			"first name",
		)
		if err != nil {
			return nil, err
		}

		user.FirstName = firstName
		hasUpdates = true
	}

	if req.LastName != nil {
		lastName, err := validation.ValidateName(
			*req.LastName,
			"last name",
		)
		if err != nil {
			return nil, err
		}

		user.LastName = lastName
		hasUpdates = true
	}

	if req.Username != nil {
		username, err := validation.ValidateUsername(*req.Username)
		if err != nil {
			return nil, err
		}

		user.Username = username
		hasUpdates = true
	}

	if req.Email != nil {
		email, err := validation.ValidateEmail(*req.Email, true)
		if err != nil {
			return nil, err
		}

		user.Email = email
		hasUpdates = true
	}

	if req.AltEmail != nil {
		altEmail, err := validation.ValidateEmail(
			*req.AltEmail,
			false,
		)
		if err != nil {
			return nil, err
		}

		user.AltEmail = altEmail
		hasUpdates = true
	}

	if req.Phone != nil {
		phone, err := validation.ValidatePhone(*req.Phone)
		if err != nil {
			return nil, err
		}

		user.Phone = phone
		hasUpdates = true
	}

	if req.DateOfBirth != nil {
		dateOfBirth, err := validation.ValidateDateOfBirth(
			*req.DateOfBirth,
		)
		if err != nil {
			return nil, err
		}

		user.DateOfBirth = dateOfBirth
		hasUpdates = true
	}

	if req.AddressLine1 != nil {
		user.AddressLine1 = strings.TrimSpace(*req.AddressLine1)
		hasUpdates = true
	}

	if req.AddressLine2 != nil {
		user.AddressLine2 = strings.TrimSpace(*req.AddressLine2)
		hasUpdates = true
	}

	if req.City != nil {
		user.City = strings.TrimSpace(*req.City)
		hasUpdates = true
	}

	if req.State != nil {
		user.State = strings.TrimSpace(*req.State)
		hasUpdates = true
	}

	if req.PinCode != nil {
		user.PinCode = strings.TrimSpace(*req.PinCode)
		hasUpdates = true
	}

	if !hasUpdates {
		return nil, ErrNoFieldsToUpdate
	}

	if user.AltEmail != "" && user.AltEmail == user.Email {
		return nil, ErrAltEmailSameAsEmail
	}

	// Primary email must not belong to another user's email/alternate email.
	if user.Email != currentUser.Email {
		existingUser, err := s.userRepository.FindByAnyEmail(
			ctx,
			user.Email,
		)
		if err == nil &&
			existingUser != nil &&
			existingUser.ID != currentUser.ID {
			return nil, ErrEmailAlreadyExists
		}

		if err != nil &&
			!errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	// Alternate email must not belong to another user's email/alternate email.
	if user.AltEmail != "" &&
		user.AltEmail != currentUser.AltEmail {
		existingUser, err := s.userRepository.FindByAnyEmail(
			ctx,
			user.AltEmail,
		)
		if err == nil &&
			existingUser != nil &&
			existingUser.ID != currentUser.ID {
			return nil, ErrAltEmailAlreadyExists
		}

		if err != nil &&
			!errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	if user.Username != currentUser.Username {
		existingUser, err := s.userRepository.FindByUsername(
			ctx,
			user.Username,
		)
		if err == nil &&
			existingUser != nil &&
			existingUser.ID != currentUser.ID {
			return nil, ErrUsernameAlreadyExists
		}

		if err != nil &&
			!errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	if user.Phone != currentUser.Phone {
		existingUser, err := s.userRepository.FindByPhone(
			ctx,
			user.Phone,
		)
		if err == nil &&
			existingUser != nil &&
			existingUser.ID != currentUser.ID {
			return nil, ErrPhoneAlreadyExists
		}

		if err != nil &&
			!errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	if err := s.userRepository.Update(ctx, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateProfilePic updates only the profile image URL/path.
func (s *UserService) UpdateProfilePic(
	ctx context.Context,
	userID string,
	fileHeader *multipart.FileHeader,
) error {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return ErrInvalidUserID
	}

	if fileHeader == nil {
		return errors.New("Profile picture is required")
	}

	const maxFileSize = 5 * 1024 * 1024

	if fileHeader.Size > maxFileSize {
		return errors.New("profile picture must not exceed 5 MB")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		// allowed
	default:
		return errors.New("profile picture must be JPEG, PNG, or WebP")
	}

	// Get the existing user so we can keep track of the old profile image.
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	oldProfilePic := strings.TrimSpace(user.ProfilePic)

	uploadDir := filepath.Join("Uploads", "profiles")

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}

	fileID := uuid.New().String()
	filename := fileID + ext
	filePath := filepath.Join(uploadDir, filename)

	// Save the new image first.
	if err := saveUploadedFile(fileHeader, filePath); err != nil {
		return err
	}

	// Update MongoDB with the new image path.
	if err := s.userRepository.UpdateProfilePic(
		ctx,
		userID,
		filePath,
	); err != nil {
		// DB update failed, so remove the newly uploaded file.
		_ = os.Remove(filePath)

		return err
	}

	// DB update succeeded, so the old image is no longer needed.
	if oldProfilePic != "" &&
		oldProfilePic != filePath &&
		isLocalProfilePic(oldProfilePic) {
		_ = os.Remove(oldProfilePic)
	}

	return nil
}

func isLocalProfilePic(path string) bool {
	cleanPath := filepath.Clean(path)
	profileDir := filepath.Clean(filepath.Join("Uploads", "profiles"))

	return strings.HasPrefix(
		cleanPath,
		profileDir+string(os.PathSeparator),
	)
}

func saveUploadedFile(
	fileHeader *multipart.FileHeader,
	destination string,
) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// Status of user account
func (s *UserService) UserStatus(
	ctx context.Context,
	adminID string,
	userID string,
	status bool,
) error {
	if adminID == userID {
		return ErrCannotChangeOwnStatus
	}

	return s.userRepository.UserStatus(
		ctx,
		userID,
		status,
	)
}

func (s *UserService) CreateCustomer(
	ctx context.Context,
	req *dto.RegisterRequest,
) (*models.User, error) {
	if req == nil {
		return nil, errors.New("create customer request is required")
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

	existingUser, err := s.userRepository.FindByAnyEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	if altEmail != "" {
		existingUser, err = s.userRepository.FindByAnyEmail(
			ctx,
			altEmail,
		)
		if err == nil && existingUser != nil {
			return nil, ErrAltEmailAlreadyExists
		}

		if err != nil &&
			!errors.Is(err, ErrUserNotFound) {
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

// ChangeUserPassword allows an administrator to reset the password
// of a customer user.
func (s *UserService) ChangeUserPassword(
	ctx context.Context,
	userID string,
	req *dto.ChangeUserPasswordRequest,
) error {

	if req == nil {
		return errors.New("change user password request is required")
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidUserID
	}

	if err := validation.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Administrators can only reset customer passwords.
	if user.Role != models.RoleCustomer {
		return ErrInvalidUserRole
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword),
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

func (s *UserService) UpdateCustomer(
	ctx context.Context,
	customerID string,
	req *dto.UpdateUserProfileRequest,
) (*models.User, error) {
	customer, err := s.userRepository.FindByID(
		ctx,
		customerID,
	)
	if err != nil {
		return nil, err
	}

	if customer.Role != models.RoleCustomer {
		return nil, ErrInvalidUserRole
	}

	return s.UpdateProfile(ctx, customerID, req)
}

func (s *UserService) ListCustomers(
	ctx context.Context,
	query dto.ListCustomersQuery,
) (*dto.CustomerListResponse, error) {
	page := query.Page
	if page == 0 {
		page = 1
	}

	if page < 1 {
		return nil, ErrInvalidPage
	}

	limit := query.Limit
	if limit == 0 {
		limit = 20
	}

	if limit < 1 || limit > 100 {
		return nil, ErrInvalidLimit
	}

	search := strings.TrimSpace(query.Search)

	customers, total, err := s.userRepository.ListCustomers(
		ctx,
		repository.CustomerListFilter{
			Status: query.Status,
			Search: search,
			Skip:   int64(page-1) * int64(limit),
			Limit:  int64(limit),
		},
	)
	if err != nil {
		return nil, err
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	return &dto.CustomerListResponse{
		Customers: customers,
		Pagination: dto.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *UserService) StartTwoFactorSetup(
	ctx context.Context,
	userID string,
) (string, string, error) {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return "", "", ErrInvalidUserID
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	if !user.Status {
		return "", "", ErrInvalidCredentials
	}

	if user.TwoFactorEnabled {
		return "", "", ErrTwoFactorAlreadyEnabled
	}

	setup, err := s.totpService.GenerateSecret(user.Username)
	if err != nil {
		return "", "", err
	}

	encryptedSecret, err := s.totpEncryptionService.Encrypt(
		setup.Secret,
	)
	if err != nil {
		return "", "", err
	}

	user.TwoFactorPendingSecret = encryptedSecret

	if err := s.userRepository.Update(ctx, user); err != nil {
		return "", "", err
	}

	return setup.Secret, setup.OTPAuthURL, nil
}

func (s *UserService) VerifyTwoFactorSetup(
	ctx context.Context,
	userID string,
	code string,
) error {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return ErrInvalidUserID
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.Status {
		return ErrInvalidCredentials
	}

	if user.TwoFactorEnabled {
		return ErrTwoFactorAlreadyEnabled
	}

	pendingSecret := strings.TrimSpace(
		user.TwoFactorPendingSecret,
	)

	if pendingSecret == "" {
		return ErrTwoFactorSetupNotStarted
	}

	decryptedSecret, err := s.totpEncryptionService.Decrypt(
		pendingSecret,
	)
	if err != nil {
		return err
	}

	if err := s.totpService.VerifyCode(
		decryptedSecret,
		code,
	); err != nil {
		return err
	}

	encryptedSecret, err := s.totpEncryptionService.Encrypt(
		decryptedSecret,
	)
	if err != nil {
		return err
	}

	user.TwoFactorSecret = encryptedSecret
	user.TwoFactorPendingSecret = ""
	user.TwoFactorEnabled = true

	if err := s.userRepository.Update(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *UserService) DisableTwoFactor(
	ctx context.Context,
	userID string,
	code string,
) error {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return ErrInvalidUserID
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.Status {
		return ErrInvalidCredentials
	}

	if !user.TwoFactorEnabled ||
		strings.TrimSpace(user.TwoFactorSecret) == "" {
		return ErrTwoFactorNotEnabled
	}

	if err := s.totpService.VerifyCode(
		user.TwoFactorSecret,
		code,
	); err != nil {
		return err
	}

	user.TwoFactorEnabled = false
	user.TwoFactorSecret = ""
	user.TwoFactorPendingSecret = ""
	user.BackupCodeHashes = nil

	return s.userRepository.Update(ctx, user)
}
