package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"basic-app/dto"
	"basic-app/models"
	"basic-app/repository"
	"basic-app/utils"
)

type PageService struct {
	pageRepository repository.PageRepository
}

func NewPageService(
	pageRepository repository.PageRepository,
) *PageService {
	return &PageService{
		pageRepository: pageRepository,
	}
}

func (s *PageService) Create(
	ctx context.Context,
	page *models.Page,
) error {
	page.Title = strings.TrimSpace(page.Title)
	page.Content = strings.TrimSpace(page.Content)

	if page.Title == "" || page.Content == "" {
		return ErrInvalidPageContent
	}

	if !isValidPageVisibility(page.Visibility) {
		return ErrInvalidVisibility
	}

	slug := utils.Slugify(page.Title)

	if slug == "" {
		return ErrInvalidSlug
	}

	slug, err := s.generateUniqueSlug(ctx, slug, "")
	if err != nil {
		return err
	}

	page.Slug = slug

	return s.pageRepository.Create(ctx, page)
}

func (s *PageService) GetByID(
	ctx context.Context,
	id string,
) (*models.Page, error) {
	return s.pageRepository.FindByID(ctx, id)
}

func (s *PageService) GetBySlug(
	ctx context.Context,
	slug string,
) (*models.Page, error) {
	slug = strings.TrimSpace(slug)

	if slug == "" {
		return nil, ErrPageNotFound
	}

	return s.pageRepository.FindBySlug(ctx, slug)
}

func (s *PageService) List(
	ctx context.Context,
	opts PageListOptions,
) ([]*models.Page, int64, int, int, error) {
	page := opts.Page

	if page == 0 {
		page = 1
	}

	if page < 1 {
		return nil, 0, 0, 0, ErrInvalidPage
	}

	limit := opts.Limit

	if limit == 0 {
		limit = 20
	}

	if limit < 1 || limit > 100 {
		return nil, 0, 0, 0, ErrInvalidLimit
	}

	if opts.Visibility != nil &&
		!isValidPageVisibility(*opts.Visibility) {
		return nil, 0, 0, 0, ErrInvalidVisibility
	}

	pages, total, err := s.pageRepository.List(
		ctx,
		repository.PageListFilter{
			Visibility: opts.Visibility,
			AuthorID:   strings.TrimSpace(opts.AuthorID),
			Skip:       int64(page-1) * int64(limit),
			Limit:      int64(limit),
		},
	)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	totalPages := 0

	if total > 0 {
		totalPages = int(
			(total + int64(limit) - 1) / int64(limit),
		)
	}

	return pages, total, page, totalPages, nil
}

func (s *PageService) Update(
	ctx context.Context,
	id string,
	req *dto.UpdatePageRequest,
) (*models.Page, error) {
	page, err := s.pageRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		page.Title = strings.TrimSpace(*req.Title)
	}

	if req.Content != nil {
		page.Content = *req.Content
	}

	if req.Visibility != nil {
		page.Visibility = *req.Visibility
	}

	if req.Slug != nil {
		slug := strings.TrimSpace(*req.Slug)

		// Empty slug means regenerate it from the current title.
		if slug == "" {
			slug = utils.Slugify(page.Title)

			if slug == "" {
				return nil, ErrInvalidSlug
			}

			// Automatically generated slug can use suffixes.
			slug, err = s.generateUniqueSlug(
				ctx,
				slug,
				page.ID.Hex(),
			)
			if err != nil {
				return nil, err
			}

			page.Slug = slug
		} else {
			// Explicitly supplied slug must not silently change.
			existingPage, err := s.pageRepository.FindBySlug(
				ctx,
				slug,
			)

			if err != nil {
				if !errors.Is(err, ErrPageNotFound) {
					return nil, err
				}
			} else {
				// Another page already owns this slug.
				if existingPage.ID.Hex() != page.ID.Hex() {
					return nil, ErrSlugAlreadyExists
				}
			}

			page.Slug = slug
		}
	}

	if page.Title == "" || page.Content == "" {
		return nil, ErrInvalidPageContent
	}

	if !isValidPageVisibility(page.Visibility) {
		return nil, ErrInvalidVisibility
	}

	if err := s.pageRepository.Update(
		ctx,
		id,
		page,
	); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *PageService) Delete(
	ctx context.Context,
	id string,
) error {
	return s.pageRepository.Delete(ctx, id)
}

func isValidPageVisibility(
	visibility models.PageVisibility,
) bool {
	return visibility == models.PageVisibilityPublic ||
		visibility == models.PageVisibilityRegistered
}

type PageListOptions struct {
	Page       int
	Limit      int
	Visibility *models.PageVisibility
	AuthorID   string
}

func (s *PageService) generateUniqueSlug(
	ctx context.Context,
	baseSlug string,
	excludeID string,
) (string, error) {
	if baseSlug == "" {
		return "", ErrInvalidSlug
	}

	slug := baseSlug

	for i := 1; ; i++ {
		page, err := s.pageRepository.FindBySlug(ctx, slug)

		if err != nil {
			if errors.Is(err, ErrPageNotFound) {
				return slug, nil
			}

			return "", err
		}

		if excludeID != "" && page.ID.Hex() == excludeID {
			return slug, nil
		}

		slug = fmt.Sprintf("%s-%d", baseSlug, i+1)
	}
}
