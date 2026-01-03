package dashboard

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DashboardWidget struct {
	ID         string                 `json:"id" bson:"id"`
	Type       string                 `json:"type" bson:"type"`
	Title      string                 `json:"title" bson:"title"`
	ModuleName string                 `json:"module_name" bson:"module_name"`
	Position   WidgetPosition         `json:"position" bson:"position"`
	Config     map[string]interface{} `json:"config" bson:"config"`
}

type WidgetPosition struct {
	X      int `json:"x" bson:"x"`
	Y      int `json:"y" bson:"y"`
	Width  int `json:"width" bson:"width"`
	Height int `json:"height" bson:"height"`
}

type DashboardConfig struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	IsDefault   bool               `json:"is_default" bson:"is_default"`
	IsShared    bool               `json:"is_shared" bson:"is_shared"`
	Widgets     []DashboardWidget  `json:"widgets" bson:"widgets"`
	Layout      string             `json:"layout" bson:"layout"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
