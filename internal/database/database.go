package database

import (
	"context"
	"log"
	"time"

	"go-crm/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"
)

// NewDatabase creates a new MongoDB database connection with lifecycle management
func NewDatabase(lc fx.Lifecycle, cfg *config.Config) (*MongodbDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB!")

	db := client.Database(cfg.DBName)

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Println("Disconnecting from MongoDB...")
			return client.Disconnect(ctx)
		},
	})

	return &MongodbDB{DB: db}, nil
}
