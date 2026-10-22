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

	// Collect unique page IDs from all menu levels.
	pageIDSet := make(map[string]struct{})
	collectMenuPageIDs(menu.Items, pageIDSet)

	pageIDs := make([]string, 0, len(pageIDSet))
	for id := range pageIDSet {
		pageIDs = append(pageIDs, id)
	}

	// Fetch all referenced pages in one database query.
	pagesByID, err := s.pageRepository.FindByIDs(ctx, pageIDs)
	if err != nil {
		return nil, err
	}

	// Build the public response using the fetched pages.
	items, err := s.buildPublicMenuItems(
		ctx,
		menu.Items,
		pagesByID,
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
	pagesByID map[string]*models.Page,
) ([]dto.PublicMenuItemResponse, error) {
	result := make(
		[]dto.PublicMenuItemResponse,
		0,
		len(items),
	)

	for _, item := range items {
		// Skip hidden items and their descendants.
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

			page, ok := pagesByID[item.PageID.Hex()]
			if !ok {
				return nil, apperrors.ErrPageNotFound
			}

			pageID := item.PageID.Hex()

			response.PageID = &pageID

			// Page URL is generated from the
			// current page slug so changing
			// the slug does not require updating
			// the menu item.
			response.URL = "/pages/" + page.Slug

		case models.MenuItemTypeURL:
			// Keep the configured URL.
			// URL may be internal or external.

		case models.MenuItemTypeGroup:
			// A group may link to a page, a URL,
			// or have no destination.
			if item.PageID != nil {
				page, ok := pagesByID[item.PageID.Hex()]
				if !ok {
					return nil, apperrors.ErrPageNotFound
				}

				pageID := item.PageID.Hex()
				response.PageID = &pageID
				response.URL = "/pages/" + page.Slug
			}

			// Preserve the group's child hierarchy,
			// regardless of whether the group has a destination.
			children, err := s.buildPublicMenuItems(
				ctx,
				item.Children,
				pagesByID,
			)
			if err != nil {
				return nil, err
			}

			response.Children = children

		default:
			return nil, apperrors.ErrInvalidMenuItem
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

		if item.URL != "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		pageID := strings.TrimSpace(*req.PageID)

		objectID, err := bson.ObjectIDFromHex(pageID)
		if err != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}

		_, err = s.pageRepository.FindByID(ctx, pageID)
		if err != nil {
			if errors.Is(err, apperrors.ErrPageNotFound) {
				return nil, apperrors.ErrPageNotFound
			}
			return nil, err
		}

		item.PageID = &objectID

	case models.MenuItemTypeURL:
		if item.URL == "" || req.PageID != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}

	case models.MenuItemTypeGroup:
		// Groups may have a URL, a Page ID, or neither.
		// They cannot have both destinations.
		if item.URL != "" && req.PageID != nil &&
			strings.TrimSpace(*req.PageID) != "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		if req.PageID != nil && strings.TrimSpace(*req.PageID) != "" {
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
		}
	}
	// Get configured maximum depth.
	maxDepthSettings, err := s.settingsService.GetMenuMaxDepth(ctx)
	if err != nil {
		return nil, err
	}

	maxDepth, err := getMenuMaxDepth(
		menu.Location,
		maxDepthSettings,
	)
	if err != nil {
		return nil, err
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
	menu, err := s.menuRepository.FindByID(ctx, menuID)
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

	// Update label.
	if req.Label != nil {
		label := strings.TrimSpace(*req.Label)
		if label == "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

		item.Label = label
	}

	// Update display status.
	if req.DisplayStatus != nil {
		item.DisplayStatus = *req.DisplayStatus
	}

	// Update type.
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
			// Destination validation is handled below.
		}
	}

	// Normalize incoming destinations.
	var pageID string
	var url string

	if req.PageID != nil {
		pageID = strings.TrimSpace(*req.PageID)
	}

	if req.URL != nil {
		url = strings.TrimSpace(*req.URL)
	}

	// A group cannot receive two non-empty destinations
	// in the same request.
	if item.Type == models.MenuItemTypeGroup &&
		req.PageID != nil &&
		req.URL != nil &&
		pageID != "" &&
		url != "" {
		return nil, apperrors.ErrInvalidMenuItem
	}

	// Update Page ID.
	if req.PageID != nil {
		switch item.Type {
		case models.MenuItemTypePage:
			if pageID == "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

		case models.MenuItemTypeURL:
			if pageID != "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

		case models.MenuItemTypeGroup:
			if pageID == "" {
				item.PageID = nil
			} else {
				// Setting a Page ID replaces the URL.
				item.URL = ""
			}
		}

		if pageID != "" {
			if item.Type != models.MenuItemTypePage &&
				item.Type != models.MenuItemTypeGroup {
				return nil, apperrors.ErrInvalidMenuItem
			}

			objectID, err := bson.ObjectIDFromHex(pageID)
			if err != nil {
				return nil, apperrors.ErrInvalidMenuItem
			}

			_, err = s.pageRepository.FindByID(ctx, pageID)
			if err != nil {
				if errors.Is(err, apperrors.ErrPageNotFound) {
					return nil, apperrors.ErrPageNotFound
				}

				return nil, err
			}

			item.PageID = &objectID
		}
	}

	// Update URL.
	if req.URL != nil {
		switch item.Type {
		case models.MenuItemTypePage:
			if url != "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

			item.URL = ""

		case models.MenuItemTypeURL:
			if url == "" {
				return nil, apperrors.ErrInvalidMenuItem
			}

			item.URL = url

		case models.MenuItemTypeGroup:
			if url != "" {
				// Setting a URL replaces the Page ID.
				item.PageID = nil
			}

			item.URL = url
		}
	}

	// Validate the final destination state.
	switch item.Type {
	case models.MenuItemTypePage:
		if item.PageID == nil || item.URL != "" {
			return nil, apperrors.ErrInvalidMenuItem
		}

	case models.MenuItemTypeURL:
		if item.URL == "" || item.PageID != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}

	case models.MenuItemTypeGroup:
		if item.URL != "" && item.PageID != nil {
			return nil, apperrors.ErrInvalidMenuItem
		}
	}

	// Get configured maximum depth.
	maxDepthSettings, err := s.settingsService.GetMenuMaxDepth(ctx)
	if err != nil {
		return nil, err
	}

	maxDepth, err := getMenuMaxDepth(
		menu.Location,
		maxDepthSettings,
	)
	if err != nil {
		return nil, err
	}

	// Groups cannot exist at the maximum depth because
	// they would have no room for children.
	if item.Type == models.MenuItemTypeGroup &&
		location.Depth >= maxDepth {
		return nil, fmt.Errorf(
			"%w: %s menu groups cannot be created at level %d",
			apperrors.ErrMenuDepthExceeded,
			menu.Location,
			location.Depth,
		)
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

func collectMenuPageIDs(
	items []models.MenuItem,
	pageIDs map[string]struct{},
) {
	for _, item := range items {
		if item.PageID != nil {
			pageIDs[item.PageID.Hex()] = struct{}{}
		}

		if len(item.Children) > 0 {
			collectMenuPageIDs(item.Children, pageIDs)
		}
	}
}
