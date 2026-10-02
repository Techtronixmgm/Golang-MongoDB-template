package mongo

import (
	"basic-app/models"
	"basic-app/services"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MenuRepository struct {
	collection *mongodriver.Collection
}

func NewMenuRepository(
	db *mongodriver.Database,
) *MenuRepository {
	return &MenuRepository{
		collection: db.Collection("menus"),
	}
}

func (r *MenuRepository) Create(
	ctx context.Context,
	menu *models.Menu,
) error {
	_, err := r.collection.InsertOne(ctx, menu)

	return err
}

func (r *MenuRepository) FindByID(
	ctx context.Context,
	id string,
) (*models.Menu, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, services.ErrMenuNotFound
	}

	var menu models.Menu

	err = r.collection.FindOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
	).Decode(&menu)

	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, services.ErrMenuNotFound
		}

		return nil, err
	}

	return &menu, nil
}

func (r *MenuRepository) FindByLocation(
	ctx context.Context,
	location models.MenuLocation,
) (*models.Menu, error) {
	var menu models.Menu

	err := r.collection.FindOne(
		ctx,
		bson.M{
			"location": location,
		},
	).Decode(&menu)

	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocuments) {
			return nil, services.ErrMenuNotFound
		}

		return nil, err
	}

	return &menu, nil
}

func (r *MenuRepository) List(
	ctx context.Context,
) ([]*models.Menu, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(
			bson.D{
				{Key: "location", Value: 1},
			},
		),
	)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	menus := make([]*models.Menu, 0)

	if err := cursor.All(ctx, &menus); err != nil {
		return nil, err
	}

	return menus, nil
}

func (r *MenuRepository) Update(
	ctx context.Context,
	id string,
	menu *models.Menu,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return services.ErrMenuNotFound
	}

	result, err := r.collection.ReplaceOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
		menu,
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrMenuNotFound
	}

	return nil
}

func (r *MenuRepository) Delete(
	ctx context.Context,
	id string,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return services.ErrMenuNotFound
	}

	result, err := r.collection.DeleteOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
	)

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return services.ErrMenuNotFound
	}

	return nil
}
