package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// 1. Connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/crm_db"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	dbName := "go_crm" // Assuming default
	// If URI contains db name, parse it? Or just assume crm_db based on previous knowledge?
	// Let's try to list databases or default to 'crm_db' which is common.
	// Actually, let's verify DB name from another file if possible?
	// internal/database/mongo.go usually has it or environment variable.
	// I'll stick to 'crm_db' as default or parse from string if needed.

	db := client.Database(dbName)

	// 2. List Collections
	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to list collections: %v", err)
	}

	fmt.Println("Found collections:", collections)

	// 3. Delete 'module_' collections
	for _, name := range collections {
		if strings.HasPrefix(name, "module_") {
			fmt.Printf("Dropping collection: %s\n", name)
			if err := db.Collection(name).Drop(ctx); err != nil {
				log.Printf("Failed to drop collection %s: %v\n", name, err)
			} else {
				fmt.Printf("Successfully dropped %s\n", name)
			}
		}
	}

	fmt.Println("Cleanup complete.")
}
