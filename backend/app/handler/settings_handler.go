package handler

import (
	"errors"
	"net/http"

	"basic-app/config"
	"basic-app/dto"
	"basic-app/services"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settingsService *services.SettingsService
}

func NewSettingsHandler(
	settingsService *services.SettingsService,
) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
	}
}

func (h *SettingsHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingsService.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get settings",
		})
		return
	}

	loginHint, err := h.settingsService.GetLoginHint(
		c.Request.Context(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get public settings",
		})
		return
	}

	c.JSON(http.StatusOK, dto.PublicSettingsResponse{
		RegistrationEnabled: settings.RegistrationEnabled,
		LoginHint:           loginHint,
	})
}

func (h *SettingsHandler) GetPrivateSettings(c *gin.Context) {
	settings, err := h.settingsService.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get settings",
		})
		return
	}

	c.JSON(http.StatusOK, dto.PrivateSettingsResponse{
		RegistrationEnabled:   settings.RegistrationEnabled,
		TwoFactorEnabled:      settings.TwoFactorEnabled,
		LoginWithPrimaryEmail: settings.LoginWithPrimaryEmail,
		LoginWithUsername:     settings.LoginWithUsername,
		LoginWithPhone:        settings.LoginWithPhone,
		LoginWithAltEmail:     settings.LoginWithAltEmail,
		MenuMaxDepth: dto.MenuMaxDepthResponse{
			Top:    settings.MenuMaxDepth.Top,
			Left:   settings.MenuMaxDepth.Left,
			Bottom: settings.MenuMaxDepth.Bottom,
		},
	})
}

func (h *SettingsHandler) Update(c *gin.Context) {
	var req dto.UpdateSettingsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.RegistrationEnabled == nil &&
		req.TwoFactorEnabled == nil &&
		req.LoginWithPrimaryEmail == nil &&
		req.LoginWithUsername == nil &&
		req.LoginWithPhone == nil &&
		req.LoginWithAltEmail == nil &&
		req.MenuMaxDepth == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no settings to update",
		})
		return
	}

	actorID := c.GetString("userID")

	if req.RegistrationEnabled != nil {
		if err := h.settingsService.UpdateRegistrationEnabled(
			c.Request.Context(),
			*req.RegistrationEnabled,
			actorID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to update registration setting",
			})
			return
		}
	}

	if req.TwoFactorEnabled != nil {
		if err := h.settingsService.UpdateTwoFactorEnabled(
			c.Request.Context(),
			*req.TwoFactorEnabled,
			actorID,
		); err != nil {
			switch {
			case errors.Is(err, config.ErrTOTPEncryptionKeyMissing):
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "2FA cannot be enabled because TOTP encryption key is not configured",
				})

			case errors.Is(err, config.ErrTOTPEncryptionKeyTooWeak):
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "2FA cannot be enabled because TOTP encryption key is too weak",
				})

			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to update two-factor setting",
				})
			}

			return
		}
	}

	if req.LoginWithPrimaryEmail != nil ||
		req.LoginWithUsername != nil ||
		req.LoginWithPhone != nil ||
		req.LoginWithAltEmail != nil {

		settings, err := h.settingsService.GetSettings(
			c.Request.Context(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to load login settings",
			})
			return
		}

		primaryEmail := settings.LoginWithPrimaryEmail
		username := settings.LoginWithUsername
		phone := settings.LoginWithPhone
		altEmail := settings.LoginWithAltEmail

		if req.LoginWithPrimaryEmail != nil {
			primaryEmail = *req.LoginWithPrimaryEmail
		}

		if req.LoginWithUsername != nil {
			username = *req.LoginWithUsername
		}

		if req.LoginWithPhone != nil {
			phone = *req.LoginWithPhone
		}

		if req.LoginWithAltEmail != nil {
			altEmail = *req.LoginWithAltEmail
		}

		if err := h.settingsService.UpdateLoginIdentifiers(
			c.Request.Context(),
			primaryEmail,
			username,
			phone,
			altEmail,
			actorID,
		); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	if req.MenuMaxDepth != nil {
		settings, err := h.settingsService.GetSettings(
			c.Request.Context(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to load menu depth settings",
			})
			return
		}

		top := settings.MenuMaxDepth.Top
		left := settings.MenuMaxDepth.Left
		bottom := settings.MenuMaxDepth.Bottom

		if req.MenuMaxDepth.Top != nil {
			top = *req.MenuMaxDepth.Top
		}

		if req.MenuMaxDepth.Left != nil {
			left = *req.MenuMaxDepth.Left
		}

		if req.MenuMaxDepth.Bottom != nil {
			bottom = *req.MenuMaxDepth.Bottom
		}

		if err := h.settingsService.UpdateMenuMaxDepth(
			c.Request.Context(),
			top,
			left,
			bottom,
			actorID,
		); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "settings updated successfully",
	})
}
