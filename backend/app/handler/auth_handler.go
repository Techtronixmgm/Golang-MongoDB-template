package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"basic-app/apperrors"
	"basic-app/config"
	"basic-app/dto"
	"basic-app/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
	userService *services.UserService
	config      config.Config
}

func NewAuthHandler(
	authService *services.AuthService,
	userService *services.UserService,
	cfg config.Config,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		config:      cfg,
	}
}

// Register handles public customer registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.authService.RegisterUser(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, apperrors.ErrUsernameAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user": gin.H{
			"id":        user.ID,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"username":  user.Username,
			"email":     user.Email,
			"altEmail":  user.AltEmail,
			"phone":     user.Phone,
			"role":      user.Role,
			"status":    user.Status,
		},
	})
}

// ChangePassword handles a user's own password change.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req dto.ChangePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	err := h.authService.ChangePassword(
		c.Request.Context(),
		userID,
		&req,
	)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidPassword):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "current password is incorrect",
			})

		case errors.Is(err, apperrors.ErrInvalidUserID):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password changed successfully",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.authService.Login(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "login failed",
		})
		return
	}

	if result.TwoFactorRequired {
		c.JSON(http.StatusOK, gin.H{
			"message":           "two-factor authentication required",
			"twoFactorRequired": true,
			"challengeToken":    result.ChallengeToken,
		})
		return
	}

	h.setAuthCookie(
		c,
		h.config.AuthAccessCookie,
		result.AccessToken,
		time.Now().UTC().Add(
			time.Duration(h.config.JWTExpiryHours)*time.Hour,
		),
	)

	h.setAuthCookie(
		c,
		h.config.AuthRefreshCookie,
		result.RefreshToken,
		result.RefreshExpiry,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":              "login successful",
		"user":                 result.User,
		"backupCodesRemaining": result.BackupCodesRemaining,
		"backupCodesLow":       result.BackupCodesLow,
	})
}

func (h *AuthHandler) setAuthCookie(
	c *gin.Context,
	name string,
	value string,
	expiresAt time.Time,
) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.config.CookieSecure,
		SameSite: cookieSameSite(h.config.CookieSameSite),
	})
}

func cookieSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(h.config.AuthRefreshCookie)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "refresh token missing",
		})
		return
	}

	result, err := h.authService.RefreshToken(
		c.Request.Context(),
		refreshToken,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidRefreshToken) {
			h.clearAuthCookies(c)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid refresh token",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to refresh token",
		})
		return
	}

	now := time.Now().UTC()

	h.setAuthCookie(
		c,
		h.config.AuthAccessCookie,
		result.AccessToken,
		now.Add(
			time.Duration(h.config.JWTExpiryHours)*time.Hour,
		),
	)

	h.setAuthCookie(
		c,
		h.config.AuthRefreshCookie,
		result.RefreshToken,
		result.RefreshExpiry,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "token refreshed successfully",
	})
}

func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	for _, name := range []string{
		h.config.AuthAccessCookie,
		h.config.AuthRefreshCookie,
	} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.config.CookieSecure,
		})
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(h.config.AuthRefreshCookie)

	if err == nil {
		err = h.authService.Logout(
			c.Request.Context(),
			refreshToken,
		)

		if err != nil &&
			!errors.Is(err, apperrors.ErrInvalidRefreshToken) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to logout",
			})
			return
		}
	}

	// Always clear browser cookies, including expired/invalid ones.
	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "logout successful",
	})
}

func (h *AuthHandler) StartTwoFactorSetup(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	secret, otpauthURL, err := h.userService.StartTwoFactorSetup(
		c.Request.Context(),
		userID,
	)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrTwoFactorAlreadyEnabled):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, apperrors.ErrInvalidUserID):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

		case errors.Is(err, apperrors.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to start two-factor setup",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "two-factor authentication setup started",
		"secret":     secret,
		"otpauthUrl": otpauthURL,
	})
}

func (h *AuthHandler) VerifyTwoFactorSetup(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "TOTP code is required",
		})
		return
	}

	backupCodes, err := h.userService.VerifyTwoFactorSetup(
		c.Request.Context(),
		userID,
		req.Code,
	)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrTwoFactorAlreadyEnabled):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, apperrors.ErrTwoFactorSetupNotStarted):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, apperrors.ErrInvalidTOTPCode):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid TOTP code",
			})

		case errors.Is(err, apperrors.ErrInvalidUserID):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

		case errors.Is(err, apperrors.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to verify two-factor setup",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "two-factor authentication enabled. Save your backup codes now. They will not be shown again. You can generate a new set later.",
		"twoFactorEnabled":     true,
		"backupCodes":          backupCodes,
		"backupCodesRemaining": len(backupCodes),
		"backupCodesLow":       len(backupCodes) <= h.config.BackupCodeLowThreshold,
	})
}

func (h *AuthHandler) VerifyTwoFactorLogin(c *gin.Context) {
	var req dto.VerifyTwoFactorLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.authService.VerifyTwoFactorLogin(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidTOTPCode) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid two-factor code",
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "two-factor authentication failed",
		})
		return
	}

	h.setAuthCookie(
		c,
		h.config.AuthAccessCookie,
		result.AccessToken,
		time.Now().UTC().Add(
			time.Duration(h.config.JWTExpiryHours)*time.Hour,
		),
	)

	h.setAuthCookie(
		c,
		h.config.AuthRefreshCookie,
		result.RefreshToken,
		result.RefreshExpiry,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":              "login successful",
		"user":                 result.User,
		"backupCodesRemaining": result.BackupCodesRemaining,
		"backupCodesLow":       result.BackupCodesLow,
	})
}

func (h *AuthHandler) DisableTwoFactor(c *gin.Context) {
	var req dto.DisableTwoFactorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	userID := c.GetString("userID")

	err := h.userService.DisableTwoFactor(
		c.Request.Context(),
		userID,
		req.Code,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidTOTPCode) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid two-factor code",
			})
			return
		}

		if errors.Is(err, apperrors.ErrTwoFactorNotEnabled) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "two-factor authentication is not enabled",
			})
			return
		}

		if errors.Is(err, apperrors.ErrInvalidUserID) ||
			errors.Is(err, apperrors.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to disable two-factor authentication",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "two-factor authentication disabled",
		"twoFactorEnabled": false,
	})
}

func (h *AuthHandler) RegenerateBackupCodes(c *gin.Context) {
	var req dto.RegenerateBackupCodesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	userID := c.GetString("userID")

	backupCodes, err := h.userService.RegenerateBackupCodes(
		c.Request.Context(),
		userID,
		req.Code,
	)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrTwoFactorNotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "two-factor authentication is not enabled",
			})

		case errors.Is(err, apperrors.ErrInvalidTOTPCode),
			errors.Is(err, apperrors.ErrInvalidBackupCode):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid two-factor code",
			})

		case errors.Is(err, apperrors.ErrInvalidUserID),
			errors.Is(err, apperrors.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to regenerate backup codes",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "backup codes regenerated successfully. Save them now. They will not be shown again.",
		"backupCodes":          backupCodes,
		"backupCodesRemaining": len(backupCodes),
		"backupCodesLow":       len(backupCodes) <= h.config.BackupCodeLowThreshold,
	})
}

func (h *AuthHandler) AdminResetTwoFactor(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user ID is required",
		})
		return
	}

	err := h.userService.ResetTwoFactor(
		c.Request.Context(),
		userID,
	)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid user ID",
			})

		case errors.Is(err, apperrors.ErrTwoFactorNotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "two-factor authentication is not enabled",
			})

		case errors.Is(err, apperrors.ErrTwoFactorResetNotAllowed):
			c.JSON(http.StatusForbidden, gin.H{
				"error": "two-factor authentication cannot be reset for admin accounts",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to reset two-factor authentication",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "two-factor authentication reset successfully",
		"twoFactorEnabled": false,
	})
}
