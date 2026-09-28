package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type RefreshToken struct {
	ID        bson.ObjectID `bson:"_id" json:"id"`
	UserID    bson.ObjectID `bson:"user_id" json:"userId"`
	TokenHash string        `bson:"token_hash" json:"-"`

	CreatedAt  time.Time  `bson:"created_at" json:"createdAt"`
	LastUsedAt *time.Time `bson:"last_used_at" json:"lastUsedAt"`
	ExpiresAt  time.Time  `bson:"expires_at" json:"expiresAt"`
	RevokedAt  *time.Time `bson:"revoked_at" json:"revokedAt"`
}
