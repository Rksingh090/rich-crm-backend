package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	app := fx.New(
		fx.Provide(
			func() *config.Config { return cfg },
			func(lc fx.Lifecycle) *fx.Lifecycle { return &lc }, // Not strictly needed to provide lifecycle, but ok.
			database.NewDatabase,                               // This uses lifecycle
			repository.NewModuleRepository,
			repository.NewRecordRepository,
		),
		fx.Invoke(seed),
	)

	// app.Run() blocks until a signal or shutdown is triggered
	app.Run()
}

func seed(lc fx.Lifecycle, shutdowner fx.Shutdowner, moduleRepo repository.ModuleRepository, recordRepo repository.RecordRepository, db *database.MongodbDB) {
	// Schedule the actual work to run after the application starts
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				// Run seeding in background
				runSeed(moduleRepo, recordRepo, db)
				// Trigger shutdown when done
				_ = shutdowner.Shutdown()
			}()
			return nil
		},
	})
}

func runSeed(moduleRepo repository.ModuleRepository, recordRepo repository.RecordRepository, db *database.MongodbDB) {
	ctx := context.Background()

	log.Println("Starting Database Seeding...")

	// 1. Fetch all modules
	modules, err := moduleRepo.List(ctx)
	if err != nil {
		log.Fatalf("Failed to list modules: %v", err)
	}
	log.Printf("Found %d modules", len(modules))

	// 2. Pre-generate IDs for all modules to support lookups
	moduleIDs := make(map[string][]primitive.ObjectID)
	recordsPerModule := 5

	for _, mod := range modules {
		ids := make([]primitive.ObjectID, recordsPerModule)
		for i := 0; i < recordsPerModule; i++ {
			ids[i] = primitive.NewObjectID()
		}
		moduleIDs[mod.Name] = ids
	}

	// 3. Seed each module
	for _, mod := range modules {
		log.Printf("Seeding module: %s...", mod.Name)

		// Drop existing collection
		collectionName := fmt.Sprintf("module_%s", mod.Name)
		if err := db.DB.Collection(collectionName).Drop(ctx); err != nil {
			log.Printf("Warning: failed to drop collection %s: %v", collectionName, err)
		}

		// Create records
		ids := moduleIDs[mod.Name]
		for i, id := range ids {
			data := generateSampleData(mod, id, moduleIDs, i)

			// We use InsertOne directly via db or repository if it supports inserting with specific ID
			// RecordRepo.Create generates a new ID usually?
			// checking RecordRepository implementation:
			// func (r *RecordRepositoryImpl) Create(..., data map[string]interface{}) ...
			// It inserts 'data'. If 'data' has "_id", Mongo driver will use it.
			// validatedData in service sets "_id". Here we set it manually.

			_, err := recordRepo.Create(ctx, mod.Name, data)
			if err != nil {
				log.Printf("Error creating record for module %s: %v", mod.Name, err)
			}
		}
	}

	log.Println("Seeding Completed Successfully!")

	// We need to stop the FX app, but since it's a CLI tool we can just exit or let it finish.
	// However, db connection is managed by lifecycle.
	// Since we are invoking 'seed' which assumes it runs during startup...
	// actually FX invokes seed and then waits. We should probably run logic and then allow shutdown.
	// But simply returning from main after app.Start will blocking.
	// Better: Use fx.Stop and app.Run? Or just handle inside seed and os.Exit(0)?
	// For simplicity, I'll just os.Exit(0) at the end of seed, forcing shutdown.
	// Or better, don't use FX for this simple script if I can avoid it, but dependencies are manageable.
}

func generateSampleData(mod models.Module, id primitive.ObjectID, moduleIDs map[string][]primitive.ObjectID, index int) map[string]interface{} {
	data := make(map[string]interface{})
	data["_id"] = id
	data["created_at"] = time.Now()
	data["updated_at"] = time.Now()

	for _, field := range mod.Fields {
		switch field.Type {
		case models.FieldTypeText, models.FieldTypeTextArea:
			data[field.Name] = fmt.Sprintf("Sample %s %d", field.Label, index+1)
		case models.FieldTypeNumber:
			data[field.Name] = float64((index + 1) * 100)
		case models.FieldTypeCurrency:
			data[field.Name] = float64((index + 1) * 150)
		case models.FieldTypeDate:
			data[field.Name] = time.Now().AddDate(0, 0, -index)
		case models.FieldTypeBoolean:
			data[field.Name] = index%2 == 0
		case models.FieldTypeEmail:
			data[field.Name] = fmt.Sprintf("user%d@example.com", index+1)
		case models.FieldTypePhone:
			data[field.Name] = fmt.Sprintf("555-010-%d", index)
		case models.FieldTypeSelect, models.FieldTypeMultiSelect:
			if len(field.Options) > 0 {
				params := rand.Perm(len(field.Options))
				if field.Type == models.FieldTypeSelect {
					data[field.Name] = field.Options[params[0]].Value
				} else {
					data[field.Name] = []string{field.Options[params[0]].Value}
				}
			}
		case models.FieldTypeLookup:
			if field.Lookup != nil {
				targetModule := field.Lookup.LookupModule
				if targetIDs, ok := moduleIDs[targetModule]; ok && len(targetIDs) > 0 {
					// Pick random ID
					targetID := targetIDs[rand.Intn(len(targetIDs))]
					data[field.Name] = targetID
				}
			}
		case models.FieldTypeFile:
			// Skip file or generate mock?
			// data[field.Name] = ""
		}
	}
	return data
}
