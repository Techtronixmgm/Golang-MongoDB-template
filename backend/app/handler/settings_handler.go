package handler

import (
	"net/http"

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

	c.JSON(http.StatusOK, dto.PublicSettingsResponse{
		RegistrationEnabled: settings.RegistrationEnabled,
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

	if req.RegistrationEnabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no settings to update",
		})
		return
	}

	actorID := c.GetString("userID")

	err := h.settingsService.UpdateRegistrationEnabled(
		c.Request.Context(),
		*req.RegistrationEnabled,
		actorID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "settings updated successfully",
	})
}
