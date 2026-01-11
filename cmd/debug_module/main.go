package main

import (
	"context"
	"fmt"
	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func CheckModule(
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

				controlDB := db.GetControlPlaneDB()
				var org map[string]interface{}
				controlDB.Collection("organizations").FindOne(ctx, bson.M{}).Decode(&org)

				if org != nil {
					tenantID := org["_id"].(primitive.ObjectID).Hex()
					fmt.Printf("Checking Tenant: %s\n", tenantID)
					tenantDB := db.GetTenantDB(tenantID)

					var mod map[string]interface{}
					err := tenantDB.Collection("modules").FindOne(ctx, bson.M{"name": "products"}).Decode(&mod)
					if err != nil {
						fmt.Printf("Module 'products' NOT FOUND: %v\n", err)
					} else {
						fmt.Printf("Module 'products' App: %v\n", mod["app"])
					}
				}
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
		fx.Invoke(CheckModule),
	)
	app.Start(context.Background())
	<-app.Done()
}
