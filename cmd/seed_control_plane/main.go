package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/core/permission"
	"go-crm/internal/core/role"
	"go-crm/internal/database"
	"go-crm/internal/features/resource"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
)

func SeedControlPlane(
	lc fx.Lifecycle,
	db *database.MongodbDB,
	shutdowner fx.Shutdowner,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer shutdowner.Shutdown()

				cpDB := db.GetControlPlaneDB()
				fmt.Println("🌱 Seeding Control Plane...")

				// 1. Seed Default Roles
				seedDefaultRoles(ctx, cpDB)

				// 2. Seed Default Modules
				seedDefaultModules(ctx, cpDB)

				// 3. Seed Default Resources
				seedDefaultResources(ctx, cpDB)

				// 4. Seed Default Permissions
				seedDefaultPermissions(ctx, cpDB)

				// 5. Seed Plans (Subscriptions)
				seedPlans(ctx, cpDB)

				fmt.Println("✅ Control Plane Seeding Complete!")
			}()
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			config.LoadConfig,
			database.NewDatabase,
		),
		fx.Invoke(SeedControlPlane),
	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	<-app.Done()
}

func seedDefaultRoles(ctx context.Context, db *mongo.Database) {
	fmt.Println("- Seeding Default Roles...")
	path := "cmd/seed/data/roles.json"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Warning: Failed to read %s: %v\n", path, err)
		return
	}

	var roles []role.Role
	if err := json.Unmarshal(data, &roles); err != nil {
		log.Fatalf("Failed to unmarshal roles: %v", err)
	}

	coll := db.Collection("default_roles")
	_ = coll.Drop(ctx)

	for _, r := range roles {
		r.ID = primitive.NewObjectID()
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
		_, _ = coll.InsertOne(ctx, r)
		fmt.Printf("  Added Role: %s\n", r.Name)
	}
}

func seedDefaultModules(ctx context.Context, db *mongo.Database) {
	fmt.Println("- Seeding Default Modules...")
	path := "cmd/seed/data/modules.json"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Warning: Failed to read %s: %v\n", path, err)
		return
	}

	var modules []models.Entity
	if err := json.Unmarshal(data, &modules); err != nil {
		log.Fatalf("Failed to unmarshal modules: %v", err)
	}

	coll := db.Collection("default_modules")
	_ = coll.Drop(ctx)

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
		"categories":             true,
		"brands":                 true,
		"products":               true,
		"vendors":                true,
		"purchase_order":         true,
		"purchase_receipts":      true,
		"shipments":              true,
		"stock_adjustments":      true,
		"stock_movements":        true,
		"inventory":              true,
		"invoices":               true,
		"invoice_items":          true,
		"purchase_invoices":      true,
		"purchase_invoice_items": true,
	}

	for i := range modules {
		if modules[i].App == "" {
			if crmModules[modules[i].Name] {
				modules[i].App = models.AppCRM
			} else if erpModules[modules[i].Name] {
				modules[i].App = models.AppERP
			} else {
				modules[i].App = models.AppCRM // Default
			}
		}

		modules[i].ID = primitive.NewObjectID()
		modules[i].CreatedAt = time.Now()
		modules[i].UpdatedAt = time.Now()
		modules[i].Scope = "global" // Templates are global
		_, _ = coll.InsertOne(ctx, modules[i])
		fmt.Printf("  Added Module: %s (App: %s)\n", modules[i].Name, modules[i].App)
	}
}

func seedDefaultResources(ctx context.Context, db *mongo.Database) {
	fmt.Println("- Seeding Default Resources...")
	path := "cmd/seed/data/resources.json"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Warning: Failed to read %s: %v\n", path, err)
		return
	}

	var resources []resource.Resource
	if err := json.Unmarshal(data, &resources); err != nil {
		log.Fatalf("Failed to unmarshal resources: %v", err)
	}

	coll := db.Collection("default_resources")
	_ = coll.Drop(ctx)

	for _, r := range resources {
		r.ID = primitive.NewObjectID()
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
		r.Scope = "global"
		_, _ = coll.InsertOne(ctx, r)
		fmt.Printf("  Added Resource: %s\n", r.ResourceID)
	}
}

type DefaultPermission struct {
	RoleName   string                 `json:"role_name" bson:"role_name"`
	App        models.App             `json:"app" bson:"app"`
	Resource   permission.ResourceRef `json:"resource" bson:"resource"`
	Actions    map[string]interface{} `json:"actions" bson:"actions"`
	FieldRules map[string]string      `json:"field_rules,omitempty" bson:"field_rules,omitempty"`
}

func seedDefaultPermissions(ctx context.Context, db *mongo.Database) {
	fmt.Println("- Seeding Default Permissions...")
	path := "cmd/seed/data/permissions.json"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Warning: Failed to read %s: %v\n", path, err)
		return
	}

	var perms []DefaultPermission
	if err := json.Unmarshal(data, &perms); err != nil {
		log.Fatalf("Failed to unmarshal permissions: %v", err)
	}

	coll := db.Collection("default_permissions")
	_ = coll.Drop(ctx)

	for _, p := range perms {
		_, _ = coll.InsertOne(ctx, p)
	}
	fmt.Printf("  Added %d template permissions\n", len(perms))
}

func seedPlans(ctx context.Context, db *mongo.Database) {
	fmt.Println("- Seeding Plans...")
	coll := db.Collection("subscriptions")
	_ = coll.Drop(ctx)

	plans := []map[string]interface{}{
		{
			"code":          "free",
			"name":          "Free Plan",
			"price":         0,
			"billing_cycle": "monthly",
			"features":      []string{"crm_basic", "erp_basic"},
		},
		{
			"code":          "pro",
			"name":          "Professional Plan",
			"price":         29.99,
			"billing_cycle": "monthly",
			"features":      []string{"crm_pro", "erp_pro", "analytics"},
		},
		{
			"code":          "enterprise",
			"name":          "Enterprise Plan",
			"price":         99.99,
			"billing_cycle": "monthly",
			"features":      []string{"all_access", "priority_support"},
		},
	}

	for _, p := range plans {
		_, _ = coll.InsertOne(ctx, p)
		fmt.Printf("  Added Plan: %s\n", p["code"])
	}
}
