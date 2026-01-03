package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DataSource represents an external or internal data source
type DataSource struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name        string                 `json:"name" bson:"name" validate:"required"`
	Description string                 `json:"description" bson:"description"`
	Type        string                 `json:"type" bson:"type" validate:"required,oneof=crm erp postgresql mysql mongodb"`
	Config      map[string]interface{} `json:"config" bson:"config"`
	IsActive    bool                   `json:"is_active" bson:"is_active"`
	CreatedBy   primitive.ObjectID     `json:"created_by" bson:"created_by"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" bson:"updated_at"`
}

// DataSourceType constants
const (
	DataSourceTypeCRM        = "crm"
	DataSourceTypeERP        = "erp"
	DataSourceTypePostgreSQL = "postgresql"
	DataSourceTypeMySQL      = "mysql"
	DataSourceTypeMongoDB    = "mongodb"
)

// Config structure examples:
// PostgreSQL/MySQL: {
//   "host": "localhost",
//   "port": 5432,
//   "database": "mydb",
//   "username": "user",
//   "password": "pass"
// }
// CRM/ERP: {} (uses internal connection)
