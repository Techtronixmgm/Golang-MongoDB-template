package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type MenuLocation string

const (
	MenuLocationTop    MenuLocation = "top"
	MenuLocationLeft   MenuLocation = "left"
	MenuLocationBottom MenuLocation = "bottom"
)

type MenuItemType string

const (
	MenuItemTypePage  MenuItemType = "page"
	MenuItemTypeURL   MenuItemType = "url"
	MenuItemTypeGroup MenuItemType = "group"
)

type Menu struct {
	ID        bson.ObjectID `bson:"_id" json:"id"`
	Name      string        `bson:"name" json:"name"`
	Location  MenuLocation  `bson:"location" json:"location"`
	Status    bool          `bson:"status" json:"status"`
	Items     []MenuItem    `bson:"items" json:"items"`
	CreatedAt time.Time     `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updatedAt"`
}

type MenuItem struct {
	ID            bson.ObjectID  `bson:"_id" json:"id"`
	Label         string         `bson:"label" json:"label"`
	Type          MenuItemType   `bson:"type" json:"type"`
	PageID        *bson.ObjectID `bson:"page_id" json:"pageId"`
	URL           string         `bson:"url" json:"url"`
	Order         int            `bson:"order" json:"order"`
	DisplayStatus bool           `bson:"display_status" json:"displayStatus"`
	Children      []MenuItem     `bson:"children" json:"children"`
}
