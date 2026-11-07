package dto

import (
	"basic-app/models"
	"time"
)

type CreatePageRequest struct {
	Title      string                `json:"title" binding:"required"`
	Content    string                `json:"content" binding:"required"`
	Visibility models.PageVisibility `json:"visibility" binding:"required"`
	CSS        string                `json:"css"`
}

type UpdatePageRequest struct {
	Title      *string                `json:"title"`
	Content    *string                `json:"content"`
	Slug       *string                `json:"slug"`
	Visibility *models.PageVisibility `json:"visibility"`
	CSS        *string                `json:"css"`
}

type ListAdminPagesQuery struct {
	Page       int    `form:"page"`
	Limit      int    `form:"limit"`
	Visibility string `form:"visibility"`
	AuthorID   string `form:"authorId"`
	Search     string `form:"search"`
}

type PageListResponse struct {
	Pages      []PageListItem `json:"pages"`
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

type UpdatePageHomepageRequest struct {
	IsHomepage *bool `json:"isHomepage"`
}

type PageListItem struct {
	ID         string                `json:"id"`
	Title      string                `json:"title"`
	Content    string                `json:"content"`
	AuthorID   string                `json:"authorId"`
	Slug       string                `json:"slug"`
	Visibility models.PageVisibility `json:"visibility"`
	IsHomepage bool                  `json:"isHomepage"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
}

func ToPageListItem(page *models.Page) PageListItem {
	return PageListItem{
		ID:         page.ID.Hex(),
		Title:      page.Title,
		Content:    page.Content,
		AuthorID:   page.AuthorID.Hex(),
		Slug:       page.Slug,
		Visibility: page.Visibility,
		IsHomepage: page.IsHomepage,
		CreatedAt:  page.CreatedAt,
		UpdatedAt:  page.UpdatedAt,
	}
}
