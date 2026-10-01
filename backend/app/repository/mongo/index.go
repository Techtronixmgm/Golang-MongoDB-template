package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Menu Indexes

func EnsureMenuIndexes(
	ctx context.Context,
	db *mongodriver.Database,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	collection := db.Collection("menus")

	indexes := []mongodriver.IndexModel{
		{
			Keys: bson.D{
				{Key: "location", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("unique_menu_location"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)

	return err
}
