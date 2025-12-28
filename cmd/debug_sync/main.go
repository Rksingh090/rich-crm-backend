package main

import (
	"context"
	"fmt"
	"go-crm/internal/config"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, _ := config.LoadConfig()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	db := client.Database(cfg.DBName)

	ctx := context.Background()

	// 1. Get Sync Setting
	fmt.Println("--- Sync Settings ---")
	opts := options.Find().SetLimit(5)
	cur, err := db.Collection("sync_settings").Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Fatal(err)
	}
	var settings []bson.M
	cur.All(ctx, &settings)
	for _, s := range settings {
		fmt.Printf("ID: %v, Name: %v, LastSyncAt: %v\n", s["_id"], s["name"], s["last_sync_at"])
		if modules, ok := s["modules"].(bson.A); ok {
			for _, m := range modules {
				if mod, ok := m.(bson.M); ok {
					fmt.Printf("  - Module: %v, SyncDeletes: %v\n", mod["module_name"], mod["sync_deletes"])
				}
			}
		}
	}

	// 2. Get Audit Logs for DELETE
	fmt.Println("\n--- Audit Logs (DELETE) ---")
	logOpts := options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(10)
	logCur, err := db.Collection("audit_logs").Find(ctx, bson.M{"action": "DELETE"}, logOpts)
	if err != nil {
		log.Fatal(err)
	}
	var logs []bson.M
	logCur.All(ctx, &logs)
	for _, l := range logs {
		fmt.Printf("ID: %v, Module: %v, Action: %v, Timestamp: %v\n", l["_id"], l["module"], l["action"], l["timestamp"])
	}
}
