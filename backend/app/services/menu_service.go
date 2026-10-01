package services

import (
	"basic-app/dto"
	"basic-app/models"
	"basic-app/repository"
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type MenuService struct {
	menuRepository repository.MenuRepository
	pageRepository repository.PageRepository
}

func NewMenuService(
	menuRepository repository.MenuRepository,
	pageRepository repository.PageRepository,
) *MenuService {
	return &MenuService{
		menuRepository: menuRepository,
		pageRepository: pageRepository,
	}
}

func (s *MenuService) Create(
	ctx context.Context,
	req *dto.CreateMenuRequest,
) (*models.Menu, error) {
	name := strings.TrimSpace(req.Name)

	if name == "" {
		return nil, ErrInvalidMenuName
	}

	if !isValidMenuLocation(req.Location) {
		return nil, ErrInvalidMenuLocation
	}

	_, err := s.menuRepository.FindByLocation(ctx, req.Location)
	if err == nil {
		return nil, ErrMenuAlreadyExists
	}

	if !errors.Is(err, ErrMenuNotFound) {
		return nil, err
	}

	now := time.Now()

	menu := &models.Menu{
		ID:       bson.NewObjectID(),
		Name:     name,
		Location: req.Location,
		Status:   req.Status,
		// Items:     make([]models.MenuItem, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.menuRepository.Create(ctx, menu); err != nil {
		return nil, err
	}

	return menu, nil
}

func (s *MenuService) Update(
	ctx context.Context,
	id string,
	req *dto.UpdateMenuRequest,
) (*models.Menu, error) {
	menu, err := s.menuRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return nil, ErrMenuNotFound
		}

		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)

		if name == "" {
			return nil, ErrInvalidMenuName
		}

		menu.Name = name
	}

	if req.Status != nil {
		menu.Status = *req.Status
	}

	menu.UpdatedAt = time.Now()

	if err := s.menuRepository.Update(
		ctx,
		id,
		menu,
	); err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return nil, ErrMenuNotFound
		}

		return nil, err
	}

	return menu, nil
}

func (s *MenuService) Get(
	ctx context.Context,
	id string,
) (*models.Menu, error) {
	menu, err := s.menuRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return nil, ErrMenuNotFound
		}

		return nil, err
	}

	return menu, nil
}

func (s *MenuService) List(
	ctx context.Context,
) ([]*models.Menu, error) {
	return s.menuRepository.List(ctx)
}

func (s *MenuService) GetPublicByLocation(
	ctx context.Context,
	location models.MenuLocation,
) (*dto.PublicMenuResponse, error) {
	if !isValidMenuLocation(location) {
		return nil, ErrInvalidMenuLocation
	}

	menu, err := s.menuRepository.FindByLocation(
		ctx,
		location,
	)
	if err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return nil, ErrMenuNotFound
		}

		return nil, err
	}

	if !menu.Status {
		return nil, ErrMenuInactive
	}

	items, err := s.buildPublicMenuItems(
		ctx,
		menu.Items,
	)
	if err != nil {
		return nil, err
	}

	return &dto.PublicMenuResponse{
		Name:     menu.Name,
		Location: menu.Location,
		Status:   menu.Status,
		Items:    items,
	}, nil
}

func (s *MenuService) buildPublicMenuItems(
	ctx context.Context,
	items []models.MenuItem,
) ([]dto.PublicMenuItemResponse, error) {
	result := make(
		[]dto.PublicMenuItemResponse,
		0,
		len(items),
	)

	for _, item := range items {

		// Hidden item is not returned.
		// For a group, this also means its
		// children are not returned.
		if !item.DisplayStatus {
			continue
		}

		response := dto.PublicMenuItemResponse{
			ID:            item.ID.Hex(),
			Label:         item.Label,
			Type:          item.Type,
			PageID:        nil,
			URL:           item.URL,
			Order:         item.Order,
			DisplayStatus: item.DisplayStatus,
			Children: make(
				[]dto.PublicMenuItemResponse,
				0,
			),
		}

		switch item.Type {

		case models.MenuItemTypePage:
			if item.PageID == nil {
				return nil, ErrInvalidMenuItem
			}

			page, err := s.pageRepository.FindByID(
				ctx,
				item.PageID.Hex(),
			)
			if err != nil {
				if errors.Is(
					err,
					ErrPageNotFound,
				) {
					return nil, ErrPageNotFound
				}

				return nil, err
			}

			pageID := item.PageID.Hex()

			response.PageID = &pageID

			// Page URL is generated from the
			// current page slug so changing
			// the slug does not require updating
			// the menu item.
			response.URL = "/pages/" + page.Slug

		case models.MenuItemTypeURL:
			// Custom URL is already stored
			// in response.URL.

		case models.MenuItemTypeGroup:
			children, err := s.buildPublicMenuItems(
				ctx,
				item.Children,
			)
			if err != nil {
				return nil, err
			}

			response.Children = children
		}

		result = append(result, response)
	}

	return result, nil
}

func isValidMenuLocation(
	location models.MenuLocation,
) bool {
	switch location {
	case models.MenuLocationTop,
		models.MenuLocationLeft,
		models.MenuLocationBottom:
		return true

	default:
		return false
	}
}

func isValidMenuItemType(
	itemType models.MenuItemType,
) bool {
	switch itemType {
	case models.MenuItemTypePage,
		models.MenuItemTypeURL,
		models.MenuItemTypeGroup:
		return true

	default:
		return false
	}
}

func (s *MenuService) Delete(
	ctx context.Context,
	id string,
) error {
	if err := s.menuRepository.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return ErrMenuNotFound
		}

		return err
	}

	return nil
}

func (s *MenuService) AddItem(
	ctx context.Context,
	id string,
	req *dto.AddMenuItemRequest,
) (*models.Menu, error) {
	menu, err := s.menuRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return nil, ErrMenuNotFound
		}

		return nil, err
	}

	label := strings.TrimSpace(req.Label)
	if label == "" {
		return nil, ErrInvalidMenuItem
	}

	if !isValidMenuItemType(req.Type) {
		return nil, ErrInvalidMenuItem
	}

	item := models.MenuItem{
		ID:            bson.NewObjectID(),
		Label:         label,
		Type:          req.Type,
		URL:           strings.TrimSpace(req.URL),
		Order:         len(menu.Items) + 1,
		DisplayStatus: true,
		Children:      make([]models.MenuItem, 0),
	}

	if req.DisplayStatus != nil {
		item.DisplayStatus = *req.DisplayStatus
	}

	switch req.Type {
	case models.MenuItemTypePage:
		if req.PageID == nil || strings.TrimSpace(*req.PageID) == "" {
			return nil, ErrInvalidMenuItem
		}

		pageID := strings.TrimSpace(*req.PageID)

		_, err := s.pageRepository.FindByID(ctx, pageID)
		if err != nil {
			if errors.Is(err, ErrPageNotFound) {
				return nil, ErrPageNotFound
			}

			return nil, err
		}

		objectID, err := bson.ObjectIDFromHex(pageID)
		if err != nil {
			return nil, ErrInvalidMenuItem
		}

		item.PageID = &objectID

		if item.URL != "" {
			return nil, ErrInvalidMenuItem
		}

	case models.MenuItemTypeURL:
		if item.URL == "" {
			return nil, ErrInvalidMenuItem
		}

		if req.PageID != nil {
			return nil, ErrInvalidMenuItem
		}

	case models.MenuItemTypeGroup:
		if item.URL != "" || req.PageID != nil {
			return nil, ErrInvalidMenuItem
		}
	}

	menu.Items = append(menu.Items, item)
	menu.UpdatedAt = time.Now()

	if err := s.menuRepository.Update(
		ctx,
		id,
		menu,
	); err != nil {
		if errors.Is(err, ErrMenuNotFound) {
			return nil, ErrMenuNotFound
		}

		return nil, err
	}

	return menu, nil
}
