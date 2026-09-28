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

const applicationSettingsCollection = "application_settings"

type ApplicationSettingsRepository struct {
	collection *mongo.Collection
}

func NewApplicationSettingsRepository(db *mongo.Database) repository.ApplicationSettingsRepository {
	return &ApplicationSettingsRepository{
		collection: db.Collection(applicationSettingsCollection),
	}
}

func (r *ApplicationSettingsRepository) Get(ctx context.Context) (*models.ApplicationSettings, error) {
	var settings models.ApplicationSettings

	err := r.collection.FindOne(
		ctx,
		bson.M{"_id": models.ApplicationSettingsID},
	).Decode(&settings)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, services.ErrApplicationSettingsNotFound
		}

		return nil, err
	}

	return &settings, nil
}

func (r *ApplicationSettingsRepository) EnsureDefaults(ctx context.Context) error {
	now := time.Now().UTC()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": models.ApplicationSettingsID},
		bson.M{
			"$setOnInsert": bson.M{
				"registration_enabled": true,
				"updated_at":           now,
				"updated_by":           "system",
			},
		},
		options.UpdateOne().SetUpsert(true),
	)

	return err
}

func (r *ApplicationSettingsRepository) UpdateRegistrationEnabled(
	ctx context.Context,
	enabled bool,
	updatedBy string,
) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": models.ApplicationSettingsID},
		bson.M{
			"$set": bson.M{
				"registration_enabled": enabled,
				"updated_at":           time.Now().UTC(),
				"updated_by":           updatedBy,
			},
		},
	)

	return err
}
