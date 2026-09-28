package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleCustomer UserRole = "customer"
)

type User struct {
	ID bson.ObjectID `bson:"_id" json:"id"`

	FirstName string `bson:"first_name" json:"firstName"`
	LastName  string `bson:"last_name" json:"lastName"`
	Username  string `bson:"username" json:"username"`
	Email     string `bson:"email" json:"email"`
	AltEmail  string `bson:"alt_email" json:"altEmail"`
	Phone     string `bson:"phone" json:"phone"`

	Role       UserRole `bson:"role" json:"role"`
	ProfilePic string   `bson:"profile_pic" json:"profilePic"`
	Status     bool     `bson:"status" json:"status"`

	PasswordHash string `bson:"password_hash" json:"-"`

	CreatedAt   time.Time  `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updatedAt"`
	LastLoginAt *time.Time `bson:"last_login_at" json:"lastLoginAt"`
	DateOfBirth *time.Time `bson:"date_of_birth" json:"dateOfBirth"`

	AddressLine1 string `bson:"address_line_1" json:"addressLine1"`
	AddressLine2 string `bson:"address_line_2" json:"addressLine2"`
	City         string `bson:"city" json:"city"`
	State        string `bson:"state" json:"state"`
	PinCode      string `bson:"pin_code" json:"pinCode"`
}

func NewCustomerUser() User {
	now := time.Now().UTC()

	return User{
		Role:      RoleCustomer,
		Status:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
