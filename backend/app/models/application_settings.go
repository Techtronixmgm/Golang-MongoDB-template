package models

import "time"

const ApplicationSettingsID = "application"

type ApplicationSettings struct {
	ID                  string    `bson:"_id" json:"-"`
	RegistrationEnabled bool      `bson:"registration_enabled" json:"registrationEnabled"`
	UpdatedAt           time.Time `bson:"updated_at" json:"-"`
	UpdatedBy           string    `bson:"updated_by" json:"-"`
}
