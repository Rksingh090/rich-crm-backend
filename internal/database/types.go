package database

import (
	"go-crm/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongodbDB struct {
	Client *mongo.Client
	Config *config.Config
	// Deprecated: existing code uses this, so we keep it for now but it points to Control Plane
	DB *mongo.Database
}

func (m *MongodbDB) GetControlPlaneDB() *mongo.Database {
	return m.Client.Database(m.Config.DBName)
}

func (m *MongodbDB) GetTenantDB(tenantID string) *mongo.Database {
	// Naming convention: tenant_<tenantID>
	return m.Client.Database("tenant_" + tenantID)
}
