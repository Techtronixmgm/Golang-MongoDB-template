package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"basic-app/apperrors"
	"basic-app/dto"
	"basic-app/models"
	"basic-app/repository"
	"basic-app/sanitizer"
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
	var err error

	page.Title = strings.TrimSpace(page.Title)

	page.Content = sanitizer.Sanitize(page.Content)
	page.Content = strings.TrimSpace(page.Content)

	css, err := sanitizer.SanitizeCSS(page.CSS)
	if err != nil {
		return err
	}
	page.CSS = css

	if page.Title == "" || page.Content == "" {
		return apperrors.ErrInvalidPageContent
	}

	if !isValidPageVisibility(page.Visibility) {
		return apperrors.ErrInvalidVisibility
	}

	slug := utils.Slugify(page.Title)

	if slug == "" {
		return apperrors.ErrInvalidSlug
	}

	slug, err = s.generateUniqueSlug(ctx, slug, "")
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
		return nil, apperrors.ErrPageNotFound
	}

	return s.pageRepository.FindBySlug(ctx, slug)
}

func (s *PageService) List(
	ctx context.Context,
	opts PageListOptions,
) ([]dto.PageListItem, int64, int, int, error) {
	page := opts.Page

	if page == 0 {
		page = 1
	}

	if page < 1 {
		return nil, 0, 0, 0, apperrors.ErrInvalidPage
	}

	limit := opts.Limit

	if limit == 0 {
		limit = 20
	}

	if limit < 1 || limit > 100 {
		return nil, 0, 0, 0, apperrors.ErrInvalidLimit
	}

	if opts.Visibility != nil &&
		!isValidPageVisibility(*opts.Visibility) {
		return nil, 0, 0, 0, apperrors.ErrInvalidVisibility
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

	pageItems := make([]dto.PageListItem, 0, len(pages))

	for _, page := range pages {
		pageItems = append(pageItems, dto.ToPageListItem(page))
	}

	return pageItems, total, page, totalPages, nil
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
		page.Content = sanitizer.Sanitize(*req.Content)
		page.Content = strings.TrimSpace(page.Content)
	}

	if req.CSS != nil {
		css, err := sanitizer.SanitizeCSS(*req.CSS)
		if err != nil {
			return nil, err
		}

		page.CSS = css
	}

	if req.Visibility != nil {
		if page.IsHomepage &&
			*req.Visibility != models.PageVisibilityPublic {
			return nil, apperrors.ErrHomepageMustBePublic
		}

		page.Visibility = *req.Visibility
	}

	if req.Slug != nil {
		slug := strings.TrimSpace(*req.Slug)

		// Empty slug means regenerate it from the current title.
		if slug == "" {
			slug = utils.Slugify(page.Title)

			if slug == "" {
				return nil, apperrors.ErrInvalidSlug
			}

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
			existingPage, err := s.pageRepository.FindBySlug(
				ctx,
				slug,
			)

			if err != nil {
				if !errors.Is(err, apperrors.ErrPageNotFound) {
					return nil, err
				}
			} else {
				if existingPage.ID.Hex() != page.ID.Hex() {
					return nil, apperrors.ErrSlugAlreadyExists
				}
			}

			page.Slug = slug
		}
	}

	if page.Title == "" || page.Content == "" {
		return nil, apperrors.ErrInvalidPageContent
	}

	if !isValidPageVisibility(page.Visibility) {
		return nil, apperrors.ErrInvalidVisibility
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
		return "", apperrors.ErrInvalidSlug
	}

	slug := baseSlug

	for i := 1; ; i++ {
		page, err := s.pageRepository.FindBySlug(ctx, slug)

		if err != nil {
			if errors.Is(err, apperrors.ErrPageNotFound) {
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

func (s *PageService) SetHomepage(
	ctx context.Context,
	id string,
	isHomepage bool,
) error {
	return s.pageRepository.SetHomepage(
		ctx,
		id,
		isHomepage,
	)
}

func (s *PageService) GetHomepage(
	ctx context.Context,
) (*models.Page, error) {
	return s.pageRepository.FindHomepage(ctx)
}
