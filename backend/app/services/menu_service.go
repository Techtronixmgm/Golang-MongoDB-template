package services

import (
	"basic-app/apperrors"
	"basic-app/dto"
	"basic-app/models"
	"basic-app/repository"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type MenuService struct {
	menuRepository  repository.MenuRepository
	pageRepository  repository.PageRepository
	settingsService *SettingsService
}

func NewMenuService(
	menuRepository repository.MenuRepository,
	pageRepository repository.PageRepository,
	settingsService *SettingsService,
) *MenuService {
	return &MenuService{
		menuRepository:  menuRepository,
		pageRepository:  pageRepository,
		settingsService: settingsService,
	}
}

func (s *MenuService) Create(
	ctx context.Context,
	req *dto.CreateMenuRequest,
) (*models.Menu, error) {
	name := strings.TrimSpace(req.Name)

	if name == "" {
		return nil, apperrors.ErrInvalidMenuName
	}

	if !isValidMenuLocation(req.Location) {
		return nil, apperrors.ErrInvalidMenuLocation
	}

	_, err := s.menuRepository.FindByLocation(ctx, req.Location)
	if err == nil {
		return nil, apperrors.ErrMenuAlreadyExists
	}

	if !errors.Is(err, apperrors.ErrMenuNotFound) {
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
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)

		if name == "" {
			return nil, apperrors.ErrInvalidMenuName
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
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
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
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
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
		return nil, apperrors.ErrInvalidMenuLocation
	}

	menu, err := s.menuRepository.FindByLocation(
		ctx,
		location,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	if !menu.Status {
		return nil, apperrors.ErrMenuInactive
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
				return nil, apperrors.ErrInvalidMenuItem
			}

			page, err := s.pageRepository.FindByID(
				ctx,
				item.PageID.Hex(),
			)
			if err != nil {
				if errors.Is(
					err,
					apperrors.ErrPageNotFound,
				) {
					return nil, apperrors.ErrPageNotFound
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
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return apperrors.ErrMenuNotFound
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
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	label := strings.TrimSpace(req.Label)

	if label == "" {
		return nil, apperrors.ErrInvalidMenuItem
	}

	if !isValidMenuItemType(req.Type) {
		return nil, apperrors.ErrInvalidMenuItem
	}

	item := models.MenuItem{
		ID:            bson.NewObjectID(),
		Label:         label,
		Type:          req.Type,
		URL:           strings.TrimSpace(req.URL),
		DisplayStatus: true,
		Children:      make([]models.MenuItem, 0),
	}

	if req.DisplayStatus != nil {
		item.DisplayStatus = *req.DisplayStatus
	}

	switch req.Type {
	case models.MenuItemTypePage:
		if req.PageID == nil || strings.TrimSpace(*req.PageID) == "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		pageID := strings.TrimSpace(*req.PageID)

		_, err := s.pageRepository.FindByID(ctx, pageID)
		if err != nil {
			if errors.Is(err, apperrors.ErrPageNotFound) {
				return nil, apperrors.ErrPageNotFound
			}

			return nil, err
		}

		objectID, err := bson.ObjectIDFromHex(pageID)
		if err != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}

		item.PageID = &objectID

		if item.URL != "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

	case models.MenuItemTypeURL:
		if item.URL == "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		if req.PageID != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}

	case models.MenuItemTypeGroup:
		if item.URL != "" || req.PageID != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}
	}

	// Get configured maximum depth.
	maxDepthSettings, err := s.settingsService.GetMenuMaxDepth(ctx)
	if err != nil {
		return nil, err
	}

	var maxDepth int

	switch menu.Location {
	case models.MenuLocationTop:
		maxDepth = maxDepthSettings.Top

	case models.MenuLocationLeft:
		maxDepth = maxDepthSettings.Left

	case models.MenuLocationBottom:
		maxDepth = maxDepthSettings.Bottom

	default:
		return nil, apperrors.ErrInvalidMenuItem
	}

	// Top-level item.
	if req.ParentID == nil || strings.TrimSpace(*req.ParentID) == "" {
		itemDepth := 1

		// Groups cannot exist at the maximum depth because
		// they would have no room for children.
		if req.Type == models.MenuItemTypeGroup &&
			itemDepth >= maxDepth {
			return nil, fmt.Errorf(
				"%w: %s menu groups cannot be created at level %d",
				apperrors.ErrMenuDepthExceeded,
				menu.Location,
				itemDepth,
			)
		}

		item.Order = len(menu.Items) + 1
		menu.Items = append(menu.Items, item)

	} else {
		parentID := strings.TrimSpace(*req.ParentID)

		parentObjectID, err := bson.ObjectIDFromHex(parentID)
		if err != nil {
			return nil, apperrors.ErrMenuItemNotFound
		}

		location, err := findMenuItemLocation(
			&menu.Items,
			parentObjectID,
			1,
		)
		if err != nil {
			if errors.Is(err, apperrors.ErrMenuItemNotFound) {
				return nil, apperrors.ErrMenuItemNotFound
			}

			return nil, err
		}

		parent := location.Item

		if parent.Type != models.MenuItemTypeGroup {
			return nil, apperrors.ErrInvalidMenuItem
		}

		itemDepth := location.Depth + 1

		// No item can be added beyond the configured maximum.
		if itemDepth > maxDepth {
			return nil, fmt.Errorf(
				"%w: %s menu cannot exceed %d levels",
				apperrors.ErrMenuDepthExceeded,
				menu.Location,
				maxDepth,
			)
		}

		// Groups cannot exist at the maximum depth because
		// they would have no room for children.
		if req.Type == models.MenuItemTypeGroup &&
			itemDepth >= maxDepth {
			return nil, fmt.Errorf(
				"%w: %s menu groups cannot be created at level %d",
				apperrors.ErrMenuDepthExceeded,
				menu.Location,
				itemDepth,
			)
		}

		item.Order = len(parent.Children) + 1

		parent.Children = append(
			parent.Children,
			item,
		)
	}

	menu.UpdatedAt = time.Now()

	if err := s.menuRepository.Update(
		ctx,
		id,
		menu,
	); err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	return menu, nil
}

func (s *MenuService) UpdateItem(
	ctx context.Context,
	menuID string,
	itemID string,
	req *dto.UpdateMenuItemRequest,
) (*models.Menu, error) {
	menu, err := s.menuRepository.FindByID(
		ctx,
		menuID,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	itemObjectID, err := bson.ObjectIDFromHex(itemID)
	if err != nil {
		return nil, apperrors.ErrMenuItemNotFound
	}

	location, err := findMenuItemLocation(
		&menu.Items,
		itemObjectID,
		1,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuItemNotFound) {
			return nil, apperrors.ErrMenuItemNotFound
		}

		return nil, err
	}

	item := location.Item

	if req.Label != nil {
		label := strings.TrimSpace(*req.Label)

		if label == "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		item.Label = label
	}

	if req.DisplayStatus != nil {
		item.DisplayStatus = *req.DisplayStatus
	}

	if req.Type != nil {
		if !isValidMenuItemType(*req.Type) {
			return nil, apperrors.ErrInvalidMenuItem
		}

		item.Type = *req.Type

		switch item.Type {
		case models.MenuItemTypePage:
			item.URL = ""

		case models.MenuItemTypeURL:
			item.PageID = nil

		case models.MenuItemTypeGroup:
			item.PageID = nil
			item.URL = ""
		}
	}

	if req.PageID != nil {
		pageID := strings.TrimSpace(*req.PageID)

		if item.Type != models.MenuItemTypePage {
			if pageID != "" {
				return nil, apperrors.ErrInvalidMenuItem
			}
		} else {
			if pageID == "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

			_, err := s.pageRepository.FindByID(
				ctx,
				pageID,
			)
			if err != nil {
				if errors.Is(err, apperrors.ErrPageNotFound) {
					return nil, apperrors.ErrPageNotFound
				}

				return nil, err
			}

			objectID, err := bson.ObjectIDFromHex(pageID)
			if err != nil {
				return nil, apperrors.ErrInvalidMenuItem
			}

			item.PageID = &objectID
		}
	}

	if req.URL != nil {
		url := strings.TrimSpace(*req.URL)

		switch item.Type {
		case models.MenuItemTypePage,
			models.MenuItemTypeGroup:
			if url != "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

			item.URL = ""

		case models.MenuItemTypeURL:
			if url == "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

			item.URL = url
		}
	}

	switch item.Type {
	case models.MenuItemTypePage:
		if item.PageID == nil {
			return nil, apperrors.ErrInvalidMenuItem
		}

		item.URL = ""

	case models.MenuItemTypeURL:
		if item.URL == "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		item.PageID = nil

	case models.MenuItemTypeGroup:
		item.PageID = nil
		item.URL = ""
	}

	menu.UpdatedAt = time.Now()

	if err := s.menuRepository.Update(
		ctx,
		menuID,
		menu,
	); err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	return menu, nil
}

func (s *MenuService) DeleteItem(
	ctx context.Context,
	menuID string,
	itemID string,
) error {
	menu, err := s.menuRepository.FindByID(
		ctx,
		menuID,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return apperrors.ErrMenuNotFound
		}

		return err
	}

	itemObjectID, err := bson.ObjectIDFromHex(itemID)
	if err != nil {
		return apperrors.ErrMenuItemNotFound
	}

	location, err := findMenuItemLocation(
		&menu.Items,
		itemObjectID,
		1,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuItemNotFound) {
			return apperrors.ErrMenuItemNotFound
		}

		return err
	}

	items := location.Items

	*items = append(
		(*items)[:location.Index],
		(*items)[location.Index+1:]...,
	)

	// Restore logical order before resequencing.
	sort.Slice(*items, func(i, j int) bool {
		return (*items)[i].Order < (*items)[j].Order
	})

	// Normalize Order values.

	for i := range *items {
		(*items)[i].Order = i + 1
	}

	menu.UpdatedAt = time.Now()

	if err := s.menuRepository.Update(
		ctx,
		menuID,
		menu,
	); err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return apperrors.ErrMenuNotFound
		}

		return err
	}

	return nil
}

func (s *MenuService) MoveItem(
	ctx context.Context,
	menuID string,
	itemID string,
	req *dto.MoveMenuItemRequest,
) (*models.Menu, error) {
	menu, err := s.menuRepository.FindByID(
		ctx,
		menuID,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	itemObjectID, err := bson.ObjectIDFromHex(itemID)
	if err != nil {
		return nil, apperrors.ErrMenuItemNotFound
	}

	location, err := findMenuItemLocation(
		&menu.Items,
		itemObjectID,
		1,
	)
	if err != nil {
		if errors.Is(err, apperrors.ErrMenuItemNotFound) {
			return nil, apperrors.ErrMenuItemNotFound
		}

		return nil, err
	}

	items := location.Items

	targetIndex := -1

	for i := range *items {
		if (*items)[i].Order == req.Order {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return nil, apperrors.ErrMenuItemOrderNotFound
	}

	if location.Index == targetIndex {
		return menu, nil
	}

	(*items)[location.Index].Order,
		(*items)[targetIndex].Order =
		(*items)[targetIndex].Order,
		(*items)[location.Index].Order

	menu.UpdatedAt = time.Now()

	if err := s.menuRepository.Update(
		ctx,
		menuID,
		menu,
	); err != nil {
		if errors.Is(err, apperrors.ErrMenuNotFound) {
			return nil, apperrors.ErrMenuNotFound
		}

		return nil, err
	}

	return menu, nil
}

func findMenuItemWithDepth(
	items []models.MenuItem,
	id bson.ObjectID,
	depth int,
) (*models.MenuItem, int, error) {
	for i := range items {
		item := &items[i]

		if item.ID == id {
			return item, depth, nil
		}

		if len(item.Children) == 0 {
			continue
		}

		found, foundDepth, err := findMenuItemWithDepth(
			item.Children,
			id,
			depth+1,
		)
		if err == nil {
			return found, foundDepth, nil
		}

		if !errors.Is(err, apperrors.ErrMenuItemNotFound) {
			return nil, 0, err
		}
	}

	return nil, 0, apperrors.ErrMenuItemNotFound
}

func findMenuItem(
	items []models.MenuItem,
	id bson.ObjectID,
	depth int,
) (*models.MenuItem, []models.MenuItem, int, error) {
	for i := range items {
		item := &items[i]

		if item.ID == id {
			return item, items, depth, nil
		}

		if len(item.Children) == 0 {
			continue
		}

		found, siblings, foundDepth, err := findMenuItem(
			item.Children,
			id,
			depth+1,
		)
		if err == nil {
			return found, siblings, foundDepth, nil
		}

		if !errors.Is(err, apperrors.ErrMenuItemNotFound) {
			return nil, nil, 0, err
		}
	}

	return nil, nil, 0, apperrors.ErrMenuItemNotFound
}

type menuItemLocation struct {
	Item  *models.MenuItem
	Items *[]models.MenuItem
	Index int
	Depth int
}

func findMenuItemLocation(
	items *[]models.MenuItem,
	id bson.ObjectID,
	depth int,
) (*menuItemLocation, error) {
	for i := range *items {
		item := &(*items)[i]

		if item.ID == id {
			return &menuItemLocation{
				Item:  item,
				Items: items,
				Index: i,
				Depth: depth,
			}, nil
		}

		if len(item.Children) == 0 {
			continue
		}

		found, err := findMenuItemLocation(
			&item.Children,
			id,
			depth+1,
		)
		if err == nil {
			return found, nil
		}

		if !errors.Is(err, apperrors.ErrMenuItemNotFound) {
			return nil, err
		}
	}

	return nil, apperrors.ErrMenuItemNotFound
}

func getMenuMaxDepth(
	location models.MenuLocation,
	settings models.MenuMaxDepthSettings,
) (int, error) {
	switch location {
	case models.MenuLocationTop:
		return settings.Top, nil

	case models.MenuLocationLeft:
		return settings.Left, nil

	case models.MenuLocationBottom:
		return settings.Bottom, nil

	default:
		return 0, apperrors.ErrInvalidMenuItem
	}
}
