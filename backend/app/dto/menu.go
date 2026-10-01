package dto

import "basic-app/models"

type CreateMenuRequest struct {
	Name     string              `json:"name" binding:"required"`
	Location models.MenuLocation `json:"location" binding:"required"`
	Status   bool                `json:"status"`
	// Items    []MenuItemRequest   `json:"items"`
}

type UpdateMenuRequest struct {
	Name   *string `json:"name"`
	Status *bool   `json:"status"`
	// Items    *[]MenuItemRequest   `json:"items"`
}
type MenuItemRequest struct {
	ID            *string              `json:"id"`
	Label         *string              `json:"label"`
	Type          *models.MenuItemType `json:"type"`
	PageID        *string              `json:"pageId"`
	URL           *string              `json:"url"`
	Order         *int                 `json:"order"`
	DisplayStatus *bool                `json:"displayStatus"`
	Children      *[]MenuItemRequest   `json:"children"`
}
type PublicMenuResponse struct {
	Name     string                   `json:"name"`
	Location models.MenuLocation      `json:"location"`
	Status   bool                     `json:"status"`
	Items    []PublicMenuItemResponse `json:"items"`
}

type PublicMenuItemResponse struct {
	ID            string                   `json:"id"`
	Label         string                   `json:"label"`
	Type          models.MenuItemType      `json:"type"`
	PageID        *string                  `json:"pageId"`
	URL           string                   `json:"url"`
	Order         int                      `json:"order"`
	DisplayStatus bool                     `json:"displayStatus"`
	Children      []PublicMenuItemResponse `json:"children"`
}
