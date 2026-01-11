package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/core/organization"
	"go-crm/internal/database"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func UpdateModules(
	lc fx.Lifecycle,
	db *database.MongodbDB,
	orgRepo organization.OrganizationRepository,
	moduleRepo module.ModuleRepository,
	recordRepo record.RecordRepository,
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

				logger.Info("🔄 Starting Module Update...")

				// 1. Load New Module Definitions
				// 1. Load New Module Definitions
				var newModules []models.Entity

				// CRM
				crmPath := "cmd/seed/data/crm_modules.json"
				b, err := os.ReadFile(crmPath)
				if err != nil {
					logger.Fatal("Failed to read crm_modules.json", zap.Error(err))
				}
				var crmModules []models.Entity
				if err := json.Unmarshal(b, &crmModules); err != nil {
					logger.Fatal("Failed to parse crm_modules.json", zap.Error(err))
				}
				newModules = append(newModules, crmModules...)

				// ERP
				erpPath := "cmd/seed/data/erp_modules.json"
				b, err = os.ReadFile(erpPath)
				if err != nil {
					logger.Fatal("Failed to read erp_modules.json", zap.Error(err))
				}
				var erpModules []models.Entity
				if err := json.Unmarshal(b, &erpModules); err != nil {
					logger.Fatal("Failed to parse erp_modules.json", zap.Error(err))
				}
				newModules = append(newModules, erpModules...)
				logger.Info("Loaded Module Definitions", zap.Int("count", len(newModules)))

				// 2. Parse CLI Args for Target Organization
				targetOrg := ""
				args := os.Args[1:]
				for _, arg := range args {
					if len(arg) > 0 && arg[0] != '-' {
						targetOrg = arg
						break
					}
				}

				if targetOrg != "" {
					logger.Info("Targeting specific organization", zap.String("org", targetOrg))
				} else {
					logger.Info("Targeting ALL organizations")
				}

				// 3. Fetch Organizations
				controlPlaneDB := db.GetControlPlaneDB()
				filter := map[string]interface{}{}
				if targetOrg != "" {
					filter["$or"] = []map[string]interface{}{
						{"name": primitive.Regex{Pattern: "^" + targetOrg + "$", Options: "i"}},
						{"slug": primitive.Regex{Pattern: "^" + targetOrg + "$", Options: "i"}},
					}
				}

				cursor, err := controlPlaneDB.Collection("organizations").Find(ctx, filter)
				if err != nil {
					logger.Fatal("Failed to list organizations", zap.Error(err))
				}
				var orgs []models.Organization
				if err := cursor.All(ctx, &orgs); err != nil {
					logger.Fatal("Failed to decode organizations", zap.Error(err))
				}

				if len(orgs) == 0 {
					logger.Warn("No organizations found matching the criteria")
					return
				}

				logger.Info("Found Organizations to Update", zap.Int("count", len(orgs)))

				// 4. Iterate Orgs
				for _, org := range orgs {
					tenantID := org.ID.Hex()
					logger.Info("Processsing Organization", zap.String("name", org.Name), zap.String("id", tenantID))

					tenantCtx := context.WithValue(ctx, models.TenantIDKey, tenantID)

					for _, modDef := range newModules {
						// Check if module exists in Tenant
						existing, err := moduleRepo.FindByName(tenantCtx, modDef.Name)

						modToSave := modDef
						modToSave.TenantID = org.ID
						modToSave.Scope = "tenant"
						modToSave.IsSystem = true
						modToSave.UpdatedAt = time.Now()
						if modToSave.App == "" {
							modToSave.App = models.AppCRM
						}

						if err == nil && existing != nil {
							// Exists - Update
							modToSave.ID = existing.ID
							modToSave.CreatedAt = existing.CreatedAt

							if err := moduleRepo.Update(tenantCtx, &modToSave); err != nil {
								logger.Error("Failed to update module", zap.String("tenant", org.Name), zap.String("module", modDef.Name), zap.Error(err))
							}
						} else {
							// Does not exist - Create
							modToSave.ID = primitive.NewObjectID()
							modToSave.CreatedAt = time.Now()

							if err := moduleRepo.Create(tenantCtx, &modToSave); err != nil {
								logger.Error("Failed to create module", zap.String("tenant", org.Name), zap.String("module", modDef.Name), zap.Error(err))
							} else {
								logger.Info("Created Missing Module", zap.String("tenant", org.Name), zap.String("module", modDef.Name))
							}
						}

						// Ensure Indexes on Record Collection (for Unique Fields)
						if err := recordRepo.EnsureIndexes(tenantCtx, &modToSave); err != nil {
							logger.Error("Failed to ensure record indexes", zap.String("module", modDef.Name), zap.Error(err))
						}
					}

					_ = moduleRepo.EnsureIndexes(tenantCtx)
				}

				logger.Info("✅ Module Update Complete (Schema + Indexes).")
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
			organization.NewOrganizationRepository,
			module.NewModuleRepository,
			record.NewRecordRepository,
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(UpdateModules),
	)

	if err := app.Start(context.Background()); err != nil {
		// logger.NewLogger(&config.Config{Level: "info"}).Fatal("App Start Failed", zap.Error(err))
	}

	<-app.Done()
}
