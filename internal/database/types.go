package database

import (
	"go-crm/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongodbDB struct {
	Client *mongo.Client
	Config *config.Config
	DB     *mongo.Database
}

func (m *MongodbDB) GetControlPlaneDB() *mongo.Database {
	return m.Client.Database(m.Config.DBName)
}

func (m *MongodbDB) GetTenantDB(tenantID string) *mongo.Database {
	// Naming convention: tenant_<tenantID>
	return m.Client.Database("tenant_" + tenantID)
}
