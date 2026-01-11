package main

import (
	"context"
	"fmt"
	"go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func FixERP(
	lc fx.Lifecycle,
	db *database.MongodbDB,
	logger *zap.Logger,
	shutdowner fx.Shutdowner,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer func() {
					_ = shutdowner.Shutdown()
				}()

				targets := []string{"products", "inventory", "orders", "invoices", "manufacturing", "supply_chain"}

				// Fetch Orgs
				controlDB := db.GetControlPlaneDB()
				cursor, err := controlDB.Collection("organizations").Find(ctx, bson.M{})
				if err != nil {
					logger.Fatal("Failed list orgs", zap.Error(err))
				}
				var orgs []models.Organization
				if err := cursor.All(ctx, &orgs); err != nil {
					logger.Fatal("Failed decode orgs", zap.Error(err))
				}

				for _, org := range orgs {
					tenantID := org.ID.Hex()
					logger.Info("Processing Org", zap.String("org", org.Name))
					tenantDB := db.GetTenantDB(tenantID)

					for _, name := range targets {
						// Update Module
						res, err := tenantDB.Collection("modules").UpdateOne(ctx,
							bson.M{"name": name},
							bson.M{"$set": bson.M{"app": "erp"}},
						)
						if err != nil {
							logger.Error("Update module failed", zap.Error(err))
						}

						if res.ModifiedCount > 0 {
							logger.Info("Moved module to ERP", zap.String("module", name))
						}

						// Update Resource
						// We update ANY resource with key=name and type=module
						_, err = tenantDB.Collection("resources").UpdateOne(ctx,
							bson.M{"key": name, "type": "module"},
							bson.M{"$set": bson.M{
								"app":         "erp",
								"resource_id": fmt.Sprintf("erp.%s", name),
							}},
						)
						if err != nil {
							// logger.Error("Update resource failed", zap.Error(err))
						}
					}
				}
				logger.Info("Done")
			}()
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			config.LoadConfig,
			logger.NewLogger,
			database.NewDatabase,
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(FixERP),
	)
	app.Start(context.Background())
	<-app.Done()
}
