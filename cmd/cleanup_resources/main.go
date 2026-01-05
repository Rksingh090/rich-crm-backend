package main

import (
	"context"
	"log"
	"time"

	"go-crm/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Simple connection without FX for the script
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	db := client.Database(cfg.DBName)
	collection := db.Collection("resources")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find duplicates
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "resource", Value: "$resource"},
				{Key: "tenant_id", Value: "$tenant_id"},
			}},
			{Key: "uniqueIds", Value: bson.D{{Key: "$addToSet", Value: "$_id"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$match", Value: bson.D{
			{Key: "count", Value: bson.D{{Key: "$gt", Value: 1}}},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	deletedCount := 0
	for cursor.Next(ctx) {
		var result struct {
			UniqueIds []interface{} `bson:"uniqueIds"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Fatal(err)
		}

		// Keep the first one, delete the rest
		for i := 1; i < len(result.UniqueIds); i++ {
			_, err := collection.DeleteOne(ctx, bson.M{"_id": result.UniqueIds[i]})
			if err != nil {
				log.Printf("Failed to delete duplicate: %v", err)
			} else {
				deletedCount++
			}
		}
	}

	log.Printf("Cleaned up %d duplicate resources.", deletedCount)

	// Drop the existing non-unique index to allow EnsureIndexes to create it with Unique=true
	log.Println("Dropping existing index idx_resource_tenant if it exists...")
	_, err = collection.Indexes().DropOne(ctx, "idx_resource_tenant")
	if err != nil {
		log.Printf("Note: Index drop returned: %v (this is fine if it didn't exist or was already dropped)", err)
	} else {
		log.Println("Successfully dropped index idx_resource_tenant")
	}
}
