package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/organization"
	"go-crm/internal/core/permission"
	"go-crm/internal/core/role"
	"go-crm/internal/core/user"
	"go-crm/internal/database"
	"go-crm/internal/features/app"
	"go-crm/internal/features/group"
	"go-crm/internal/features/module"
	"go-crm/internal/features/resource"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Seed runs the database seeding
func Seed(
	lc fx.Lifecycle,
	moduleRepo module.ModuleRepository,
	resourceService resource.ResourceService,
	appService app.AppService,
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
				modulesPath := "cmd/seed/data/modules.json"
				resourcesPath := "cmd/seed/data/resources.json"
				appsPath := "cmd/seed/data/apps.json"

				// 0. Seed Apps (Global)
				var apps []common_models.Application
				if err := readJSON(appsPath, &apps); err != nil {
					logger.Warn("Failed to read apps.json, skipping app seeding", zap.Error(err))
				} else {
					if err := appService.SyncApps(ctx, apps); err != nil {
						logger.Error("Failed to sync global apps", zap.Error(err))
					} else {
						logger.Info("Global Apps synced successfully", zap.Int("count", len(apps)))
					}
				}

				// 1. Seed Resources (Global)
				var resources []resource.Resource
				if err := readJSON(resourcesPath, &resources); err != nil {
					logger.Warn("Failed to read resources.json, skipping resource seeding", zap.Error(err))
				} else {
					// Enforce Global Scope
					for i := range resources {
						resources[i].Scope = "global"
						resources[i].CanOverride = true
						// Ensure TenantID is empty/nil for global (json unmarshal leaves it empty usually)
					}

					// Use background context (no tenant)
					if err := resourceService.SyncResources(ctx, resources); err != nil {
						logger.Error("Failed to sync global resources", zap.Error(err))
					} else {
						logger.Info("Global Resources synced successfully", zap.Int("count", len(resources)))
					}
				}

				// 2. Seed Modules (Global Schema)
				var modules []common_models.Entity
				if err := readJSON(modulesPath, &modules); err != nil {
					logger.Fatal("Failed to read modules.json", zap.Error(err))
				}

				// Product Mapping
				crmModules := map[string]bool{
					"accounts":      true,
					"contacts":      true,
					"leads":         true,
					"opportunities": true,
					"tasks":         true,
					"meetings":      true,
					"calls":         true,
				}
				erpModules := map[string]bool{
					"products":               true,
					"categories":             true,
					"brands":                 true,
					"tax_rates":              true,
					"price_lists":            true,
					"price_list_items":       true,
					"customers":              true,
					"vendors":                true,
					"invoices":               true,
					"invoice_items":          true,
					"purchase_invoices":      true,
					"purchase_invoice_items": true,
				}

				for i := range modules {
					// Set App
					if modules[i].App == "" {
						if crmModules[modules[i].Name] {
							modules[i].App = common_models.AppCRM
						} else if erpModules[modules[i].Name] {
							modules[i].App = common_models.AppERP
						} else {
							modules[i].App = common_models.AppCRM // Default
						}
					}

					// Enforce Global Scope
					modules[i].Scope = "global"
					modules[i].CanOverride = true
					modules[i].IsSystem = true // Seeded modules are system modules

					modules[i].ID = primitive.NewObjectID()
					modules[i].CreatedAt = time.Now()
					modules[i].UpdatedAt = time.Now()

					if err := moduleRepo.Create(ctx, &modules[i]); err != nil {
						// If it exists, try to update it to ensure app field is populated
						if errUpdate := moduleRepo.Update(ctx, &modules[i]); errUpdate != nil {
							logger.Warn("Failed to update module (might be missing context)", zap.String("module", modules[i].Name))
						} else {
							logger.Info("Global Module updated", zap.String("module", modules[i].Name), zap.String("app", string(modules[i].App)))
						}
					} else {
						logger.Info("Global Module created", zap.String("module", modules[i].Name), zap.String("app", string(modules[i].App)))
					}
				}

				logger.Info("✅ Seeding Complete! (Global Resources & Modules Only)")
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
			role.NewRoleRepository,
			role.NewRoleService,
			user.NewUserRepository,
			fx.Annotate(
				user.NewUserRepository,
				fx.As(new(audit.UserFinder)),
			),
			module.NewModuleRepository,
			group.NewGroupRepository,
			organization.NewOrganizationRepository,
			resource.NewResourceRepository,
			resource.NewResourceService,
			app.NewAppRepository,
			app.NewAppService,
			permission.NewPermissionRepository,
			audit.NewAuditRepository,
			audit.NewAuditService,
			permission.NewPermissionService,
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
