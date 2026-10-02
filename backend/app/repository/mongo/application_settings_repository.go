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
		bson.A{
			bson.M{
				"$set": bson.M{
					"registration_enabled": bson.M{
						"$ifNull": bson.A{
							"$registration_enabled",
							true,
						},
					},
					"two_factor_enabled": bson.M{
						"$ifNull": bson.A{
							"$two_factor_enabled",
							false,
						},
					},
					"login_with_primary_email": bson.M{
						"$ifNull": bson.A{
							"$login_with_primary_email",
							true,
						},
					},
					"login_with_username": bson.M{
						"$ifNull": bson.A{
							"$login_with_username",
							true,
						},
					},
					"login_with_phone": bson.M{
						"$ifNull": bson.A{
							"$login_with_phone",
							false,
						},
					},
					"login_with_alt_email": bson.M{
						"$ifNull": bson.A{
							"$login_with_alt_email",
							false,
						},
					},
					"updated_at": bson.M{
						"$ifNull": bson.A{
							"$updated_at",
							now,
						},
					},
					"updated_by": bson.M{
						"$ifNull": bson.A{
							"$updated_by",
							"system",
						},
					},

					"menu_max_depth": bson.M{
						"$ifNull": bson.A{
							"$menu_max_depth",
							bson.M{
								"top":    3,
								"left":   1,
								"bottom": 2,
							},
						},
					},
				},
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

func (r *ApplicationSettingsRepository) UpdateTwoFactorEnabled(
	ctx context.Context,
	enabled bool,
	updatedBy string,
) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": models.ApplicationSettingsID},
		bson.M{
			"$set": bson.M{
				"two_factor_enabled": enabled,
				"updated_at":         time.Now().UTC(),
				"updated_by":         updatedBy,
			},
		},
	)

	return err
}

func (r *ApplicationSettingsRepository) UpdateLoginIdentifiers(
	ctx context.Context,
	primaryEmail bool,
	username bool,
	phone bool,
	altEmail bool,
	updatedBy string,
) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": models.ApplicationSettingsID},
		bson.M{
			"$set": bson.M{
				"login_with_primary_email": primaryEmail,
				"login_with_username":      username,
				"login_with_phone":         phone,
				"login_with_alt_email":     altEmail,
				"updated_at":               time.Now().UTC(),
				"updated_by":               updatedBy,
			},
		},
	)

	return err
}

func (r *ApplicationSettingsRepository) UpdateMenuMaxDepth(
	ctx context.Context,
	top int,
	left int,
	bottom int,
	updatedBy string,
) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": models.ApplicationSettingsID},
		bson.M{
			"$set": bson.M{
				"menu_max_depth.top":    top,
				"menu_max_depth.left":   left,
				"menu_max_depth.bottom": bottom,
				"updated_at":            time.Now().UTC(),
				"updated_by":            updatedBy,
			},
		},
	)

	return err
}
