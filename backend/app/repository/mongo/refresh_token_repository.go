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
)

const refreshTokenCollection = "refresh_tokens"

type RefreshTokenRepository struct {
	collection *mongo.Collection
}

func NewRefreshTokenRepository(
	db *mongo.Database,
) repository.RefreshTokenRepository {
	return &RefreshTokenRepository{
		collection: db.Collection(refreshTokenCollection),
	}
}

func (r *RefreshTokenRepository) Create(
	ctx context.Context,
	token *models.RefreshToken,
) error {
	if token.ID.IsZero() {
		token.ID = bson.NewObjectID()
	}

	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	_, err := r.collection.InsertOne(ctx, token)
	return err
}

func (r *RefreshTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*models.RefreshToken, error) {
	var token models.RefreshToken

	err := r.collection.FindOne(
		ctx,
		bson.M{
			"token_hash": tokenHash,
			"revoked_at": nil,
			"expires_at": bson.M{"$gt": time.Now().UTC()},
		},
	).Decode(&token)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, services.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	return &token, nil
}

func (r *RefreshTokenRepository) Touch(ctx context.Context, tokenID string) error {
	objectID, err := bson.ObjectIDFromHex(tokenID)
	if err != nil {
		return services.ErrRefreshTokenNotFound
	}

	now := time.Now().UTC()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":        objectID,
			"revoked_at": nil,
		},
		bson.M{
			"$set": bson.M{
				"last_used_at": now,
			},
		},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrRefreshTokenNotFound
	}

	return nil
}

func (r *RefreshTokenRepository) Revoke(
	ctx context.Context,
	tokenID string,
) error {
	objectID, err := bson.ObjectIDFromHex(tokenID)
	if err != nil {
		return services.ErrRefreshTokenNotFound
	}

	now := time.Now().UTC()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":        objectID,
			"revoked_at": nil,
		},
		bson.M{
			"$set": bson.M{
				"revoked_at": now,
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return services.ErrRefreshTokenNotFound
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeAllByUserID(
	ctx context.Context,
	userID string,
) error {
	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return services.ErrRefreshTokenNotFound
	}

	now := time.Now().UTC()

	_, err = r.collection.UpdateMany(
		ctx,
		bson.M{
			"user_id":    objectID,
			"revoked_at": nil,
		},
		bson.M{
			"$set": bson.M{
				"revoked_at": now,
			},
		},
	)

	return err
}

func (r *RefreshTokenRepository) DeleteExpired(
	ctx context.Context,
	before time.Time,
) error {
	_, err := r.collection.DeleteMany(
		ctx,
		bson.M{
			"expires_at": bson.M{
				"$lt": before,
			},
		},
	)

	return err
}
