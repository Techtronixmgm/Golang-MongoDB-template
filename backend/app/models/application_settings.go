package models

import (
	"time"
)

const ApplicationSettingsID = "application"

type ApplicationSettings struct {
	ID                    string    `bson:"_id" json:"-"`
	RegistrationEnabled   bool      `bson:"registration_enabled" json:"registrationEnabled"`
	TwoFactorEnabled      bool      `bson:"two_factor_enabled" json:"twoFactorEnabled"`
	LoginWithPrimaryEmail bool      `bson:"login_with_primary_email" json:"loginWithPrimaryEmail"`
	LoginWithUsername     bool      `bson:"login_with_username" json:"loginWithUsername"`
	LoginWithPhone        bool      `bson:"login_with_phone" json:"loginWithPhone"`
	LoginWithAltEmail     bool      `bson:"login_with_alt_email" json:"loginWithAltEmail"`
	UpdatedAt             time.Time `bson:"updated_at" json:"-"`
	UpdatedBy             string    `bson:"updated_by" json:"-"`
}
