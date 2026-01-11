package main

import (
	"context"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func CreateAthlon(
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
				orgColl := controlDB.Collection("organizations")
				userColl := controlDB.Collection("users")

				// 1. Create Organization
				orgName := "Athlon"
				orgSlug := "athlon"
				var org models.Organization

				err := orgColl.FindOne(ctx, bson.M{"slug": orgSlug}).Decode(&org)
				if err != nil {
					logger.Info("Creating Organization", zap.String("name", orgName))
					org = models.Organization{
						ID:                 primitive.NewObjectID(),
						Name:               orgName,
						Slug:               orgSlug,
						Plan:               "enterprise",
						SubscriptionStatus: models.SubscriptionStatusActive,
						CreatedAt:          time.Now(),
						EnabledApps:        []models.App{models.AppCRM, models.AppERP},
					}
					if _, err := orgColl.InsertOne(ctx, org); err != nil {
						logger.Fatal("Failed to create org", zap.Error(err))
					}
				} else {
					logger.Info("Organization exists", zap.String("id", org.ID.Hex()))
				}

				// 2. Create Admin User
				email := "admin@athlon.com"
				var user models.User
				err = userColl.FindOne(ctx, bson.M{"email": email}).Decode(&user)
				if err != nil {
					logger.Info("Creating Admin User", zap.String("email", email))
					hashed, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

					user = models.User{
						ID:              primitive.NewObjectID(),
						Email:           email,
						Password:        string(hashed),
						FirstName:       "Admin",
						LastName:        "Athlon",
						Status:          "active",
						TenantID:        org.ID,
						AppRoles:        []models.UserAppRole{},
						IsPlatformAdmin: true,
						CreatedAt:       time.Now(),
					}
					if _, err := userColl.InsertOne(ctx, user); err != nil {
						logger.Fatal("Failed to create user", zap.Error(err))
					}
				} else {
					logger.Info("User exists", zap.String("id", user.ID.Hex()))
				}

				// 3. Mark Org Owner
				orgColl.UpdateOne(ctx, bson.M{"_id": org.ID}, bson.M{"$set": bson.M{"owner_id": user.ID}})

				logger.Info("✅ Athlon Created.")
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
		fx.Invoke(CreateAthlon),
	)
	app.Start(context.Background())
	<-app.Done()
}
