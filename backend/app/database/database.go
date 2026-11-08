package database

import (
	"basic-app/config"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {

	//prevents your app freeze in startup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoUri)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect failed: %w", err)
	}
	//ping check
	if err := client.Ping(ctx, nil); err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer disconnectCancel()

		if disconnectErr := client.Disconnect(disconnectCtx); disconnectErr != nil {
			log.Printf("mongo disconnect after failed ping: %v", disconnectErr)
		}

		return nil, nil, fmt.Errorf("mongo ping failed: %w", err)
	}

	db := client.Database(cfg.MongoDB)

	log.Printf("MongoDB connected (database: %s)", cfg.MongoDB)

	return client, db, nil
}

func Disconnect(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}
