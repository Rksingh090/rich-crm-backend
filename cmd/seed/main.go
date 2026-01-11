package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"go-crm/internal/features/record"
	"go-crm/internal/features/resource"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Seed runs the database seeding
func Seed(
	lc fx.Lifecycle,
	db *database.MongodbDB,
	moduleRepo module.ModuleRepository,
	resourceService resource.ResourceService,
	appService app.AppService,
	orgRepo organization.OrganizationRepository,
	userRepo user.UserRepository,
	recordRepo record.RecordRepository,
	roleService role.RoleService,
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

				// --- 1. CLEANUP (Remove earlier data) ---
				controlDB := db.GetControlPlaneDB()
				collectionsToDrop := []string{
					"default_modules",
					"default_resources",
					"apps",
					"organizations",
					"users",
					// Cleanup potential garbage from previous runs/bugs
					"modules",
					"resources",
					"roles",
					"permissions",
				}
				for _, collName := range collectionsToDrop {
					if err := controlDB.Collection(collName).Drop(ctx); err != nil {
						logger.Warn("Failed to drop collection", zap.String("collection", collName), zap.Error(err))
					} else {
						logger.Info("Dropped collection", zap.String("collection", collName))
					}
				}

				// Helper to read JSON
				readJSON := func(path string, v interface{}) error {
					b, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					return json.Unmarshal(b, v)
				}

				// Data Paths
				crmModulesPath := "cmd/seed/data/crm_modules.json"
				erpModulesPath := "cmd/seed/data/erp_modules.json"
				resourcesPath := "cmd/seed/data/resources.json"
				appsPath := "cmd/seed/data/apps.json"

				// --- 2. SEED METADATA ---

				// 2a. Seed Apps (Global)
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

				// 2b. Seed Resources (Global)
				var resources []resource.Resource
				if err := readJSON(resourcesPath, &resources); err != nil {
					logger.Warn("Failed to read resources.json, skipping resource seeding", zap.Error(err))
				} else {
					for i := range resources {
						resources[i].Scope = "global"
						resources[i].CanOverride = true
					}
					if err := resourceService.SyncResources(ctx, resources); err != nil {
						logger.Error("Failed to sync global resources", zap.Error(err))
					} else {
						logger.Info("Global Resources synced successfully", zap.Int("count", len(resources)))
					}
				}

				// 2c. Seed Modules (Global Schema)
				var modules []common_models.Entity

				var crmModulesList []common_models.Entity
				if err := readJSON(crmModulesPath, &crmModulesList); err != nil {
					logger.Fatal("Failed to read crm_modules.json", zap.Error(err))
				}
				modules = append(modules, crmModulesList...)

				var erpModulesList []common_models.Entity
				if err := readJSON(erpModulesPath, &erpModulesList); err != nil {
					logger.Fatal("Failed to read erp_modules.json", zap.Error(err))
				}
				modules = append(modules, erpModulesList...)
				// Product Mapping
				crmModules := map[string]bool{
					"accounts": true, "contacts": true, "leads": true,
					"opportunities": true, "tasks": true, "meetings": true, "calls": true,
				}
				erpModules := map[string]bool{
					"products": true, "categories": true, "brands": true,
					"tax_rates": true, "price_lists": true, "customers": true,
					"vendors": true, "invoices": true, "purchase_invoices": true,
				}

				for i := range modules {
					if modules[i].App == "" {
						if crmModules[modules[i].Name] {
							modules[i].App = common_models.AppCRM
						} else if erpModules[modules[i].Name] {
							modules[i].App = common_models.AppERP
						} else {
							modules[i].App = common_models.AppCRM
						}
					}
					modules[i].Scope = "global"
					modules[i].CanOverride = true
					modules[i].IsSystem = true
					modules[i].ID = primitive.NewObjectID()
					modules[i].CreatedAt = time.Now()
					modules[i].UpdatedAt = time.Now()

					if err := moduleRepo.Create(ctx, &modules[i]); err != nil {
						// Ignore dup error if any, or update
						if mongo.IsDuplicateKeyError(err) {
							_ = moduleRepo.Update(ctx, &modules[i])
						} else {
							logger.Error("Failed to create global module", zap.String("module", modules[i].Name), zap.Error(err))
						}
					}
				}
				logger.Info("Global Modules synced successfully", zap.Int("count", len(modules)))

				// --- 3. SEED DEMO TENANT & DATA ---

				// 3a. Create Tenant
				tenantID := primitive.NewObjectID()
				demoOrg := &common_models.Organization{
					ID:                 tenantID,
					Name:               "Demo Corp",
					Slug:               "demo-corp",
					Plan:               "enterprise",
					SubscriptionStatus: common_models.SubscriptionStatusActive,
					Currency:           "USD",
					ValidationStatus:   common_models.ValidationStatusVerified,
					EnabledApps:        []common_models.App{common_models.AppCRM, common_models.AppERP},
					CreatedAt:          time.Now(),
					UpdatedAt:          time.Now(),
				}

				// Check if exists
				existingOrg, err := orgRepo.FindByName(ctx, "Demo Corp")
				if err == nil && existingOrg != nil {
					logger.Info("Demo Tenant already exists, using existing.", zap.String("id", existingOrg.ID.Hex()))
					tenantID = existingOrg.ID
					demoOrg = existingOrg
				} else {
					if err := orgRepo.Create(ctx, demoOrg); err != nil {
						logger.Error("Failed to create demo tenant", zap.Error(err))
						return
					}
					logger.Info("Created Demo Tenant", zap.String("id", tenantID.Hex()))
				}

				// --- 3x. CLEANUP TENANT DATA (Reset Tenant Modules) ---
				tenantDB := db.GetTenantDB(tenantID.Hex())
				tenantDeletes := []string{"modules", "resources", "roles", "permissions", "crm_leads", "crm_contacts", "crm_tasks"}
				for _, collName := range tenantDeletes {
					_ = tenantDB.Collection(collName).Drop(ctx)
					logger.Info("Dropped Tenant Collection", zap.String("tenant", "demo"), zap.String("collection", collName))
				}

				// 3b. Create User (Admin)
				userID := primitive.NewObjectID()
				adminUser := &common_models.User{
					ID:        userID,
					TenantID:  tenantID,
					Email:     "admin@demo.com",
					FirstName: "Admin",
					LastName:  "User",
					Status:    "active",
					// Password: "password" (hashed ideally, skipping hash logic for simplicity or using auth service if available)
					// In real scenario, use authService.Signup
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
					IsPlatformAdmin: true,
					Password:        "password", // Plain text for now as per AuthServiceImpl
				}
				// Mock password (bcrypt cost 10 for "password")
				// $2a$10$X7h.m/FzQ/x.P.P.P.P.P.P.P.P.P.P.P. (dummy)
				// Actually let's assume no auth or simple check for now.

				// Context with Tenant
				tenantCtx := context.WithValue(ctx, common_models.TenantIDKey, tenantID.Hex())

				existingUser, err := userRepo.FindByEmail(tenantCtx, adminUser.Email)
				if err == nil && existingUser != nil {
					logger.Info("Admin User already exists", zap.String("id", existingUser.ID.Hex()))
					userID = existingUser.ID
				} else {
					// Need to ensure userRepo.Create uses tenant ID from context
					if err := userRepo.Create(tenantCtx, adminUser); err != nil {
						logger.Error("Failed to create admin user", zap.Error(err))
					} else {
						logger.Info("Created Admin User", zap.String("email", adminUser.Email))
					}
				}

				// 3c. Seed Sample Records (Leads & Contacts)
				logger.Info("Seeding Sample Records...")

				// Context with User ID for CreatedBy
				recordCtx := context.WithValue(tenantCtx, "user_id", userID.Hex())

				// Leads
				leads := []map[string]interface{}{
					{"first_name": "John", "last_name": "Doe", "email": "john@example.com", "company": "Acme Inc", "status": "New"},
					{"first_name": "Jane", "last_name": "Smith", "email": "jane@test.com", "company": "Test Corp", "status": "Contacted"},
					{"first_name": "Bob", "last_name": "Brown", "email": "bob@start.up", "company": "Startup Hub", "status": "Qualified"},
					{"first_name": "Alice", "last_name": "Green", "email": "alice@nature.org", "company": "Nature LLC", "status": "New"},
					{"first_name": "Charlie", "last_name": "White", "email": "charlie@cloud.net", "company": "Cloud Sys", "status": "Lost"},
				}

				for _, data := range leads {
					// Use RecordRepo directly to bypass complex service checks if needed
					// But ensure we target 'leads' module and 'crm' app
					if _, err := recordRepo.Create(recordCtx, "leads", common_models.AppCRM, data); err != nil {
						logger.Warn("Failed to create sample lead", zap.String("email", fmt.Sprintf("%v", data["email"])), zap.Error(err))
					}
				}
				logger.Info("Seeded Leads")

				// Contacts
				contacts := []map[string]interface{}{
					{"first_name": "Sarah", "last_name": "Connor", "email": "sarah@sky.net", "phone": "555-0101"},
					{"first_name": "Kyle", "last_name": "Reese", "email": "kyle@resistance.org", "phone": "555-0102"},
					{"first_name": "Tony", "last_name": "Stark", "email": "tony@stark.com", "phone": "555-0199"},
				}

				for _, data := range contacts {
					if _, err := recordRepo.Create(recordCtx, "contacts", common_models.AppCRM, data); err != nil {
						logger.Warn("Failed to create sample contact", zap.String("email", fmt.Sprintf("%v", data["email"])), zap.Error(err))
					}
				}
				logger.Info("Seeded Contacts")

				logger.Info("✅ Seeding Complete! Demo environment ready.")
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
			database.NewDatabase, // Returns *MongodbDB
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

			// Added for Record Seeding
			record.NewRecordRepository,
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
