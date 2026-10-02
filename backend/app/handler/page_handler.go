package handler

import (
	"errors"
	"net/http"
	"strings"

	"basic-app/apperrors"
	"basic-app/dto"
	"basic-app/models"
	"basic-app/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PageHandler struct {
	pageService *services.PageService
}

func NewPageHandler(
	pageService *services.PageService,
) *PageHandler {
	return &PageHandler{
		pageService: pageService,
	}
}

func (h *PageHandler) Create(c *gin.Context) {
	var req dto.CreatePageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	authorID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	page := &models.Page{
		Title:      req.Title,
		Content:    req.Content,
		AuthorID:   authorID,
		Visibility: req.Visibility,
	}

	if err := h.pageService.Create(
		c.Request.Context(),
		page,
	); err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidPageContent),
			errors.Is(err, apperrors.ErrInvalidVisibility),
			errors.Is(err, apperrors.ErrInvalidSlug):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to create page",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "page created successfully",
		"page":    page,
	})
}

func (h *PageHandler) GetBySlug(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "page slug is required",
		})
		return
	}

	page, err := h.pageService.GetBySlug(
		c.Request.Context(),
		slug,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrPageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "page not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to fetch page",
		})
		return
	}

	if page.Visibility == models.PageVisibilityRegistered {
		userID := strings.TrimSpace(c.GetString("userID"))

		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"page": page,
	})
}

func (h *PageHandler) List(c *gin.Context) {
	var query dto.ListAdminPagesQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid query parameters",
		})
		return
	}

	var visibility *models.PageVisibility

	if query.Visibility != "" {
		v := models.PageVisibility(query.Visibility)
		visibility = &v
	}

	limit := query.Limit
	if limit == 0 {
		limit = 20
	}

	pages, total, page, totalPages, err := h.pageService.List(
		c.Request.Context(),
		services.PageListOptions{
			Page:       query.Page,
			Limit:      query.Limit,
			Visibility: visibility,
			AuthorID:   query.AuthorID,
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidPage):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, apperrors.ErrInvalidLimit):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, apperrors.ErrInvalidVisibility):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to list pages",
			})
		}
		return
	}

	c.JSON(http.StatusOK, dto.PageListResponse{
		Pages: pages,
		Pagination: dto.PagePagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *PageHandler) Update(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "page id is required",
		})
		return
	}

	var req dto.UpdatePageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	page, err := h.pageService.Update(
		c.Request.Context(),
		id,
		&req,
	)

	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrPageNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "page not found",
			})

		case errors.Is(err, apperrors.ErrSlugAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": "slug already exists",
			})

		case errors.Is(err, apperrors.ErrInvalidPageContent),
			errors.Is(err, apperrors.ErrInvalidVisibility),
			errors.Is(err, apperrors.ErrInvalidSlug):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unable to update page",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "page updated successfully",
		"page":    page,
	})
}

func (h *PageHandler) Delete(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "page id is required",
		})
		return
	}

	if err := h.pageService.Delete(
		c.Request.Context(),
		id,
	); err != nil {
		if errors.Is(err, apperrors.ErrPageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "page not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to delete page",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "page deleted successfully",
	})
}
