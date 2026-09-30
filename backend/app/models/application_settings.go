package models

import "time"

const ApplicationSettingsID = "application"

type ApplicationSettings struct {
	ID                  string    `bson:"_id" json:"-"`
	RegistrationEnabled bool      `bson:"registration_enabled" json:"registrationEnabled"`
	TwoFactorEnabled    bool      `bson:"two_factor_enabled" json:"twoFactorEnabled"`
	UpdatedAt           time.Time `bson:"updated_at" json:"-"`
	UpdatedBy           string    `bson:"updated_by" json:"-"`
}
