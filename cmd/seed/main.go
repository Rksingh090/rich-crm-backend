package main

import (
	"context"
	"log"
	"time"

	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/logger"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Seed runs the database seeding
func Seed(
	lc fx.Lifecycle,
	roleRepo repository.RoleRepository,
	userRepo repository.UserRepository,
	logger *zap.Logger,
	shutdowner fx.Shutdowner,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer func() {
					if err := shutdowner.Shutdown(); err != nil {
						logger.Error("Failed to shutdown", zap.Error(err))
					}
				}()

				logger.Info("🌱 Starting Database Seeding...")

				// 1. Seed Roles
				roles := []models.Role{
					{
						Name:        "admin",
						Description: "Administrator with full access",
						IsSystem:    true,
						ModulePermissions: map[string]models.ModulePermission{
							"*": {Create: true, Read: true, Update: true, Delete: true}, // Wildcard
						},
					},
					{
						Name:        "manager",
						Description: "Manager with access to most modules",
						IsSystem:    true,
						ModulePermissions: map[string]models.ModulePermission{
							"users":   {Create: true, Read: true, Update: true, Delete: false},
							"reports": {Create: true, Read: true, Update: true, Delete: true},
						},
					},
					{
						Name:        "user",
						Description: "Standard user",
						IsSystem:    true,
						ModulePermissions: map[string]models.ModulePermission{
							"profile": {Create: false, Read: true, Update: true, Delete: false},
						},
					},
				}

				createdRoles := make(map[string]primitive.ObjectID)

				for _, role := range roles {
					existing, err := roleRepo.FindByName(ctx, role.Name)
					if err == nil {
						logger.Info("Role exists, skipping", zap.String("role", role.Name))
						createdRoles[role.Name] = existing.ID
						continue
					}

					role.ID = primitive.NewObjectID()
					role.CreatedAt = time.Now()
					role.UpdatedAt = time.Now()

					if err := roleRepo.Create(ctx, &role); err != nil {
						logger.Error("Failed to create role", zap.String("role", role.Name), zap.Error(err))
						continue
					}
					logger.Info("Role created", zap.String("role", role.Name))
					createdRoles[role.Name] = role.ID
				}

				// 2. Seed Admin User
				adminUsername := "admin"
				_, err := userRepo.FindByUsername(ctx, adminUsername)
				if err == nil {
					logger.Info("Admin user exists, skipping")
				} else {
					adminRoleID, ok := createdRoles["admin"]
					if !ok {
						// Fallback check if it was skipped
						existing, err := roleRepo.FindByName(ctx, "admin")
						if err == nil {
							adminRoleID = existing.ID
						} else {
							logger.Error("Admin role not found, cannot create admin user")
							return
						}
					}

					adminUser := models.User{
						ID:        primitive.NewObjectID(),
						Username:  adminUsername,
						Password:  "admin123", // Plaintext for now as per current AuthServiceImpl
						Email:     "admin@example.com",
						FirstName: "System",
						LastName:  "Admin",
						Status:    "active",
						Roles:     []primitive.ObjectID{adminRoleID},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}

					if err := userRepo.Create(ctx, &adminUser); err != nil {
						logger.Error("Failed to create admin user", zap.Error(err))
					} else {
						logger.Info("Admin user created", zap.String("username", adminUsername), zap.String("password", "admin123"))
					}
				}

				logger.Info("✅ Seeding Complete!")
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
			repository.NewRoleRepository,
			repository.NewUserRepository,
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(Seed),
	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	<-app.Done()
}
