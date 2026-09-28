package mongo

import (
	"context"
	"errors"
	"time"

	"basic-app/models"
	"basic-app/repository"
	"basic-app/services"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const pageCollection = "pages"

type PageRepository struct {
	collection *mongo.Collection
}

func NewPageRepository(db *mongo.Database) repository.PageRepository {
	return &PageRepository{
		collection: db.Collection(pageCollection),
	}
}

func (r *PageRepository) Create(
	ctx context.Context,
	page *models.Page,
) error {
	if page.ID.IsZero() {
		page.ID = bson.NewObjectID()
	}

	now := time.Now().UTC()

	if page.CreatedAt.IsZero() {
		page.CreatedAt = now
	}

	page.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, page)
	return err
}

func (r *PageRepository) FindByID(
	ctx context.Context,
	id string,
) (*models.Page, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, services.ErrPageNotFound
	}

	var page models.Page

	err = r.collection.FindOne(
		ctx,
		bson.M{"_id": objectID},
	).Decode(&page)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, services.ErrPageNotFound
		}

		return nil, err
	}

	return &page, nil
}

func (r *PageRepository) FindBySlug(
	ctx context.Context,
	slug string,
) (*models.Page, error) {
	var page models.Page

	err := r.collection.FindOne(
		ctx,
		bson.M{"slug": slug},
	).Decode(&page)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, services.ErrPageNotFound
		}

		return nil, err
	}

	return &page, nil
}

func (r *PageRepository) List(
	ctx context.Context,
	filter repository.PageListFilter,
) ([]*models.Page, int64, error) {
	query := bson.M{}

	if filter.Visibility != nil {
		query["visibility"] = *filter.Visibility
	}

	if filter.AuthorID != "" {
		authorID, err := bson.ObjectIDFromHex(filter.AuthorID)
		if err != nil {
			return nil, 0, err
		}

		query["author_id"] = authorID
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	cursor, err := r.collection.Find(
		ctx,
		query,
		options.Find().
			SetSkip(filter.Skip).
			SetLimit(filter.Limit).
			SetSort(bson.D{
				{Key: "created_at", Value: -1},
			}),
	)

	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var pages []*models.Page

	if err := cursor.All(ctx, &pages); err != nil {
		return nil, 0, err
	}

	return pages, total, nil
}

func (r *PageRepository) Update(
	ctx context.Context,
	id string,
	page *models.Page,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return services.ErrPageNotFound
	}

	page.UpdatedAt = time.Now().UTC()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"title":      page.Title,
				"content":    page.Content,
				"slug":       page.Slug,
				"visibility": page.Visibility,
				"updated_at": page.UpdatedAt,
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrPageNotFound
	}

	return nil
}

func (r *PageRepository) Delete(
	ctx context.Context,
	id string,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return services.ErrPageNotFound
	}

	result, err := r.collection.DeleteOne(
		ctx,
		bson.M{"_id": objectID},
	)

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return services.ErrPageNotFound
	}

	return nil
}
