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
						Password:  "Rishu@090", // Plaintext for now as per current AuthServiceImpl
						Email:     "root@gocrm.com",
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

				// 3. Seed System Modules
				systemModules := []models.Module{
					{
						Name:     "accounts",
						Label:    "Accounts",
						IsSystem: true,
						Fields: []models.ModuleField{
							{Name: "name", Label: "Account Name", Type: models.FieldTypeText, Required: true},
							{Name: "industry", Label: "Industry", Type: models.FieldTypeSelect, Required: false, Options: []models.SelectOptions{{Label: "Tech", Value: "Tech"}, {Label: "Finance", Value: "Finance"}, {Label: "Retail", Value: "Retail"}, {Label: "Manufacturing", Value: "Manufacturing"}, {Label: "Healthcare", Value: "Healthcare"}}},
							{Name: "website", Label: "Website", Type: models.FieldTypeURL, Required: false},
							{Name: "phone", Label: "Phone", Type: models.FieldTypePhone, Required: false},
							{Name: "type", Label: "Type", Type: models.FieldTypeSelect, Required: false, Options: []models.SelectOptions{{Label: "Customer", Value: "Customer"}, {Label: "Partner", Value: "Partner"}, {Label: "Vendor", Value: "Vendor"}}},
						},
					},
					{
						Name:     "contacts",
						Label:    "Contacts",
						IsSystem: true,
						Fields: []models.ModuleField{
							{Name: "first_name", Label: "First Name", Type: models.FieldTypeText, Required: true},
							{Name: "last_name", Label: "Last Name", Type: models.FieldTypeText, Required: true},
							{Name: "email", Label: "Email", Type: models.FieldTypeEmail, Required: true},
							{Name: "phone", Label: "Phone", Type: models.FieldTypePhone, Required: false},
							{Name: "account", Label: "Account", Type: models.FieldTypeLookup, Required: false, Lookup: &models.LookupDef{LookupModule: "accounts", LookupLabel: "name", ValueField: "_id"}},
							{Name: "title", Label: "Job Title", Type: models.FieldTypeText, Required: false},
						},
					},
					{
						Name:     "leads",
						Label:    "Leads",
						IsSystem: true,
						Fields: []models.ModuleField{
							{Name: "name", Label: "Full Name", Type: models.FieldTypeText, Required: true},
							{Name: "email", Label: "Email", Type: models.FieldTypeEmail, Required: true},
							{Name: "phone", Label: "Phone", Type: models.FieldTypePhone, Required: false},
							{Name: "company", Label: "Company", Type: models.FieldTypeText, Required: false},
							{Name: "status", Label: "Status", Type: models.FieldTypeSelect, Required: true, Options: []models.SelectOptions{{Label: "New", Value: "New"}, {Label: "Contacted", Value: "Contacted"}, {Label: "Qualified", Value: "Qualified"}, {Label: "Lost", Value: "Lost"}}},
							{Name: "source", Label: "Source", Type: models.FieldTypeSelect, Required: false, Options: []models.SelectOptions{{Label: "Web", Value: "Web"}, {Label: "Referral", Value: "Referral"}, {Label: "Event", Value: "Event"}}},
						},
					},
					{
						Name:     "opportunities",
						Label:    "Opportunities",
						IsSystem: true,
						Fields: []models.ModuleField{
							{Name: "name", Label: "Opportunity Name", Type: models.FieldTypeText, Required: true},
							{Name: "amount", Label: "Amount", Type: models.FieldTypeCurrency, Required: true},
							{Name: "stage", Label: "Stage", Type: models.FieldTypeSelect, Required: true, Options: []models.SelectOptions{{Label: "Prospecting", Value: "Prospecting"}, {Label: "Negotiation", Value: "Negotiation"}, {Label: "Closed Won", Value: "Closed Won"}, {Label: "Closed Lost", Value: "Closed Lost"}}},
							{Name: "close_date", Label: "Close Date", Type: models.FieldTypeDate, Required: true},
							{Name: "account", Label: "Account", Type: models.FieldTypeLookup, Required: false, Lookup: &models.LookupDef{LookupModule: "accounts", LookupLabel: "name", ValueField: "_id"}},
						},
					},
					{
						Name:     "products",
						Label:    "Products",
						IsSystem: true,
						Fields: []models.ModuleField{
							{Name: "name", Label: "Product Name", Type: models.FieldTypeText, Required: true},
							{Name: "code", Label: "Product Code", Type: models.FieldTypeText, Required: true},
							{Name: "price", Label: "Price", Type: models.FieldTypeCurrency, Required: true},
							{Name: "description", Label: "Description", Type: models.FieldTypeTextArea, Required: false},
							{Name: "category", Label: "Category", Type: models.FieldTypeSelect, Required: false, Options: []models.SelectOptions{{Label: "Software", Value: "Software"}, {Label: "Hardware", Value: "Hardware"}, {Label: "Service", Value: "Service"}}},
						},
					},
					{
						Name:     "sales",
						Label:    "Sales Orders",
						IsSystem: true,
						Fields: []models.ModuleField{
							{Name: "order_number", Label: "Order Number", Type: models.FieldTypeText, Required: true},
							{Name: "account", Label: "Customer", Type: models.FieldTypeLookup, Required: true, Lookup: &models.LookupDef{LookupModule: "accounts", LookupLabel: "name", ValueField: "_id"}},
							{Name: "amount", Label: "Total Amount", Type: models.FieldTypeCurrency, Required: true},
							{Name: "order_date", Label: "Order Date", Type: models.FieldTypeDate, Required: true},
							{Name: "status", Label: "Status", Type: models.FieldTypeSelect, Required: true, Options: []models.SelectOptions{{Label: "Pending", Value: "Pending"}, {Label: "Completed", Value: "Completed"}, {Label: "Cancelled", Value: "Cancelled"}}},
						},
					},
				}

				for _, module := range systemModules {
					existing, err := moduleRepo.FindByName(ctx, module.Name)
					if err == nil {
						logger.Info("Module exists, skipping", zap.String("module", module.Name))
						// Determine if we should update System flag if it was manually created?
						// For now, respect existing. But if it exists without IsSystem, maybe we should set it?
						// Let's safe-play: just skip. User might have created custom 'leads' module.
						// Or, force update IsSystem=true if names match standard?
						if !existing.IsSystem {
							existing.IsSystem = true
							_ = moduleRepo.Update(ctx, existing)
							logger.Info("Marked existing module as system", zap.String("module", module.Name))
						}
						continue
					}

					module.ID = primitive.NewObjectID()
					module.CreatedAt = time.Now()
					module.UpdatedAt = time.Now()

					if err := moduleRepo.Create(ctx, &module); err != nil {
						logger.Error("Failed to create system module", zap.String("module", module.Name), zap.Error(err))
					} else {
						logger.Info("System module created", zap.String("module", module.Name))
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
