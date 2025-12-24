package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Permission struct {
	Code        string `bson:"code" json:"code"` // e.g., "user:create"
	Description string `bson:"description" json:"description"`
}

type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Permissions []string           `bson:"permissions" json:"permissions"` // List of Permission codes
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type User struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Username  string               `bson:"username" json:"username"`
	Password  string               `bson:"password" json:"-"`
	Email     string               `bson:"email" json:"email"`
	FirstName string               `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName  string               `bson:"last_name,omitempty" json:"last_name,omitempty"`
	Phone     string               `bson:"phone,omitempty" json:"phone,omitempty"`
	Status    string               `bson:"status" json:"status"` // active, inactive, suspended
	Roles     []primitive.ObjectID `bson:"roles" json:"roles"`   // References to Role IDs
	LastLogin *time.Time           `bson:"last_login,omitempty" json:"last_login,omitempty"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}
