package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type PageVisibility string

const (
	PageVisibilityPublic     PageVisibility = "public"
	PageVisibilityRegistered PageVisibility = "registered"
)

type Page struct {
	ID         bson.ObjectID  `bson:"_id" json:"id"`
	Title      string         `bson:"title" json:"title"`
	Content    string         `bson:"content" json:"content"`
	AuthorID   bson.ObjectID  `bson:"author_id" json:"authorId"`
	Slug       string         `bson:"slug" json:"slug"`
	Visibility PageVisibility `bson:"visibility" json:"visibility"`

	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`
}
