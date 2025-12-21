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
	Password  string               `bson:"password" json:"password"`
	Email     string               `bson:"email" json:"email"`
	Roles     []primitive.ObjectID `bson:"roles" json:"roles"` // References to Role IDs
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}
