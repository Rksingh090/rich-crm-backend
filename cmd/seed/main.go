package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
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
	moduleRepo repository.ModuleRepository,
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

				logger.Info("🌱 Starting Database Seeding from JSON...")

				// Helper to read JSON
				readJSON := func(path string, v interface{}) error {
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					return json.Unmarshal(b, v)
				}

				// Data Paths (Assuming running from backend root)
				rolesPath := "cmd/seed/data/roles.json"
				usersPath := "cmd/seed/data/users.json"
				modulesPath := "cmd/seed/data/modules.json"

				// 1. Seed Roles
				var roles []models.Role
				if err := readJSON(rolesPath, &roles); err != nil {
					logger.Fatal("Failed to read roles.json", zap.Error(err))
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

				// 2. Seed Users
				var usersData []struct {
					Username  string   `json:"username"`
					Password  string   `json:"password"`
					Email     string   `json:"email"`
					FirstName string   `json:"first_name"`
					LastName  string   `json:"last_name"`
					Status    string   `json:"status"`
					RoleNames []string `json:"roles"`
				}
				if err := readJSON(usersPath, &usersData); err != nil {
					logger.Error("Failed to read users.json", zap.Error(err))
					// Don't fatal, maybe we can seed modules
				} else {
					for _, uData := range usersData {
						_, err := userRepo.FindByUsername(ctx, uData.Username)
						if err == nil {
							logger.Info("User exists, skipping", zap.String("username", uData.Username))
							continue
						}

						// Map role names to IDs
						var roleIDs []primitive.ObjectID
						for _, rName := range uData.RoleNames {
							if rID, ok := createdRoles[rName]; ok {
								roleIDs = append(roleIDs, rID)
							} else {
								// Try finding it
								r, err := roleRepo.FindByName(ctx, rName)
								if err == nil {
									roleIDs = append(roleIDs, r.ID)
								} else {
									logger.Warn("Role found in user definition but not in DB", zap.String("role", rName))
								}
							}
						}

						newUser := models.User{
							ID:        primitive.NewObjectID(),
							Username:  uData.Username,
							Password:  uData.Password, // Ideally hash this if AuthServiceImpl expects hashed, but current seeding used plaintext?
							Email:     uData.Email,
							FirstName: uData.FirstName,
							LastName:  uData.LastName,
							Status:    uData.Status,
							Roles:     roleIDs,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}

						if err := userRepo.Create(ctx, &newUser); err != nil {
							logger.Error("Failed to create user", zap.String("username", uData.Username), zap.Error(err))
						} else {
							logger.Info("User created", zap.String("username", uData.Username))
						}
					}
				}

				// 3. Seed Modules
				var modules []models.Module
				if err := readJSON(modulesPath, &modules); err != nil {
					logger.Fatal("Failed to read modules.json", zap.Error(err))
				}

				for _, module := range modules {
					existing, err := moduleRepo.FindByName(ctx, module.Name)
					if err == nil {
						logger.Info("Module exists, checking for field updates", zap.String("module", module.Name))

						// Merge fields
						updated := false
						existingFieldMap := make(map[string]bool)
						for _, f := range existing.Fields {
							existingFieldMap[f.Name] = true
						}

						for _, newField := range module.Fields {
							if !existingFieldMap[newField.Name] {
								existing.Fields = append(existing.Fields, newField)
								logger.Info("Adding new field to module", zap.String("module", module.Name), zap.String("field", newField.Name))
								updated = true
							}
						}

						if updated {
							existing.UpdatedAt = time.Now()
							if err := moduleRepo.Update(ctx, existing); err != nil {
								logger.Error("Failed to update module", zap.String("module", module.Name), zap.Error(err))
							} else {
								logger.Info("Module updated", zap.String("module", module.Name))
							}
						}
						continue
					}

					module.ID = primitive.NewObjectID()
					module.CreatedAt = time.Now()
					module.UpdatedAt = time.Now()

					if err := moduleRepo.Create(ctx, &module); err != nil {
						logger.Error("Failed to create module", zap.String("module", module.Name), zap.Error(err))
					} else {
						logger.Info("Module created", zap.String("module", module.Name))
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
			repository.NewModuleRepository,
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
