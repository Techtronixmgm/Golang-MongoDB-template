package dto

import (
	"basic-app/models"
	"time"
)

type CreatePageRequest struct {
	Title      string                `json:"title" binding:"required"`
	Content    string                `json:"content" binding:"required"`
	Visibility models.PageVisibility `json:"visibility" binding:"required"`
}

type UpdatePageRequest struct {
	Title      *string                `json:"title"`
	Content    *string                `json:"content"`
	Slug       *string                `json:"slug"`
	Visibility *models.PageVisibility `json:"visibility"`
}

type ListAdminPagesQuery struct {
	Page       int    `form:"page"`
	Limit      int    `form:"limit"`
	Visibility string `form:"visibility"`
	AuthorID   string `form:"authorId"`
}

type PageListResponse struct {
	Pages      []*models.Page `json:"pages"`
	Pagination PagePagination `json:"pagination"`
}

type PagePagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type PublicPageResponse struct {
	ID         string                `json:"id"`
	Title      string                `json:"title"`
	Content    string                `json:"content"`
	Slug       string                `json:"slug"`
	Visibility models.PageVisibility `json:"visibility"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
}

func ToPublicPageResponse(page *models.Page) PublicPageResponse {
	return PublicPageResponse{
		ID:         page.ID.Hex(),
		Title:      page.Title,
		Content:    page.Content,
		Slug:       page.Slug,
		Visibility: page.Visibility,
		CreatedAt:  page.CreatedAt,
		UpdatedAt:  page.UpdatedAt,
	}
}
