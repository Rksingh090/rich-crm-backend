package resource

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceUI struct {
	Sidebar    bool   `bson:"sidebar" json:"sidebar"`
	Order      int    `bson:"order" json:"order"`
	Group      string `bson:"group" json:"group"`
	GroupOrder int    `bson:"group_order" json:"group_order"`
	Location   string `bson:"location" json:"location"`
}

type Resource struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ResourceID   string             `bson:"resource" json:"resource_id"`
	TenantID     primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	Product      string             `bson:"product" json:"product"`
	Type         string             `bson:"type" json:"type"`
	Key          string             `bson:"key" json:"key"`
	Label        string             `bson:"label" json:"label"`
	Icon         string             `bson:"icon" json:"icon"`
	Route        string             `bson:"route" json:"route"`
	Actions      []string           `bson:"actions" json:"actions"`
	Configurable bool               `bson:"configurable" json:"configurable"`
	UI           ResourceUI         `bson:"ui" json:"ui"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
