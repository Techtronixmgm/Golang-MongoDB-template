package dto

import (
	"basic-app/models"
	"time"
)

// RegisterRequest contains the allowed fields
// for public customer registration.
type RegisterRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`

	Username string `json:"username"`

	Email    string `json:"email"`
	AltEmail string `json:"altEmail"`
	Phone    string `json:"phone"`

	Password string `json:"password"`
}

// ChangePasswordRequest contains the current password
// and the new password for a password change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type LoginResult struct {
	User              *models.User
	AccessToken       string
	RefreshToken      string
	RefreshExpiry     time.Time
	TwoFactorRequired bool
	ChallengeToken    string

	BackupCodesRemaining int
	BackupCodesLow       bool
}

type RefreshResult struct {
	AccessToken   string
	RefreshToken  string
	RefreshExpiry time.Time
}

type VerifyTwoFactorLoginRequest struct {
	ChallengeToken string `json:"challengeToken"`
	Code           string `json:"code"`
}

type DisableTwoFactorRequest struct {
	Code string `json:"code"`
}

type RegenerateBackupCodesRequest struct {
	Code string `json:"code"`
}
