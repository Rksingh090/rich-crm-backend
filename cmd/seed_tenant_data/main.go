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
	"go-crm/internal/features/record"
	"go-crm/internal/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Data Structures for JSON Decoding
type DemoData struct {
	Categories []map[string]interface{} `json:"categories"`
	ERP        struct {
		Vendors  []map[string]interface{} `json:"vendors"`
		Products []map[string]interface{} `json:"products"`
	} `json:"erp_data"`
	CRM struct {
		Accounts      []map[string]interface{} `json:"accounts"`
		Contacts      []map[string]interface{} `json:"contacts"`
		Leads         []map[string]interface{} `json:"leads"`
		Opportunities []map[string]interface{} `json:"opportunities"`
		Tasks         []map[string]interface{} `json:"tasks"`
		Meetings      []map[string]interface{} `json:"meetings"`
		Calls         []map[string]interface{} `json:"calls"`
	} `json:"crm_data"`
	Orders struct {
		SalesOrders    []map[string]interface{} `json:"sales_orders"`
		PurchaseOrders []map[string]interface{} `json:"purchase_orders"`
	} `json:"orders"`
}

func SeedTenantData(
	lc fx.Lifecycle,
	db *database.MongodbDB,
	orgRepo organization.OrganizationRepository,
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

				logger.Info("🌱 Starting Tenant Data Seeding from JSON...")

				// 1. Load JSON Data
				dataPath := "cmd/seed/data/demo_data.json"
				b, err := os.ReadFile(dataPath)
				if err != nil {
					logger.Fatal("Failed to read demo_data.json", zap.Error(err))
				}
				var data DemoData
				if err := json.Unmarshal(b, &data); err != nil {
					logger.Fatal("Failed to parse demo_data.json", zap.Error(err))
				}
				logger.Info("Loaded JSON Data")

				// 2. Parse Args & Find Org
				targetOrg := "athlon" // Default
				if len(os.Args) > 1 {
					targetOrg = os.Args[1]
				}

				controlPlaneDB := db.GetControlPlaneDB()
				var org models.Organization
				filter := bson.M{
					"$or": []bson.M{
						{"name": primitive.Regex{Pattern: "^" + targetOrg + "$", Options: "i"}},
						{"slug": primitive.Regex{Pattern: "^" + targetOrg + "$", Options: "i"}},
					},
				}
				if err := controlPlaneDB.Collection("organizations").FindOne(ctx, filter).Decode(&org); err != nil {
					logger.Fatal("Organization not found", zap.String("org", targetOrg), zap.Error(err))
				}

				// 3. Setup Context
				var adminUser models.User
				userIDStr := ""
				if err := controlPlaneDB.Collection("users").FindOne(ctx, bson.M{"tenant_id": org.ID}).Decode(&adminUser); err == nil {
					userIDStr = adminUser.ID.Hex()
				}
				tenantID := org.ID.Hex()
				tenantCtx := context.WithValue(ctx, models.TenantIDKey, tenantID)
				if userIDStr != "" {
					tenantCtx = context.WithValue(tenantCtx, "user_id", userIDStr)
				}
				logger.Info("Seeding for Org", zap.String("name", org.Name))

				// --- CLEANUP OLD DATA ---
				logger.Info("🧹 Dropping Old Collections...")
				tenantDB := db.GetTenantDB(tenantID)
				collsToDrop := []string{
					"crm_leads", "crm_contacts", "crm_accounts", "crm_opportunities", "crm_tasks", "crm_calls", "crm_meetings",
					"erp_products", "erp_vendors", "erp_categories", "erp_sales_orders", "erp_purchase_orders", "erp_inventory_adjustments",
				}
				for _, c := range collsToDrop {
					_ = tenantDB.Collection(c).Drop(ctx)
				}
				// -------------------------

				// Maps for ID Lookup
				accountMap := make(map[string]primitive.ObjectID)
				contactMap := make(map[string]primitive.ObjectID) // key: email
				productMap := make(map[string]primitive.ObjectID)
				categoryMap := make(map[string]primitive.ObjectID)
				vendorMap := make(map[string]primitive.ObjectID)

				// Helper
				create := func(module string, app models.App, item map[string]interface{}) (primitive.ObjectID, error) {
					res, err := recordRepo.Create(tenantCtx, module, app, item)
					if err != nil {
						return primitive.NilObjectID, err
					}
					if oid, ok := res.(primitive.ObjectID); ok {
						return oid, nil
					}
					return primitive.NilObjectID, nil
				}

				// --- 1. Categories ---
				for _, d := range data.Categories {
					if oid, err := create("categories", models.AppERP, d); err == nil {
						if name, ok := d["name"].(string); ok {
							categoryMap[name] = oid
						}
					}
				}
				logger.Info("Seeded Categories")

				// --- 2. ERP: Vendors & Products ---
				for _, d := range data.ERP.Vendors {
					if oid, err := create("vendors", models.AppERP, d); err == nil {
						if name, ok := d["name"].(string); ok {
							vendorMap[name] = oid
						}
					}
				}
				for _, d := range data.ERP.Products {
					catName, _ := d["category_name"].(string)
					delete(d, "category_name")
					if catID, ok := categoryMap[catName]; ok {
						d["category"] = catID
					}
					if oid, err := create("products", models.AppERP, d); err == nil {
						if name, ok := d["name"].(string); ok {
							productMap[name] = oid
						}
					}
				}
				logger.Info("Seeded ERP Basics")

				// --- 3. CRM: Accounts, Contacts, Leads ---
				for _, d := range data.CRM.Accounts {
					if oid, err := create("accounts", models.AppCRM, d); err == nil {
						if name, ok := d["name"].(string); ok {
							accountMap[name] = oid
						}
					}
				}
				for _, d := range data.CRM.Contacts {
					acctName, _ := d["account_name"].(string)
					delete(d, "account_name")
					if acctID, ok := accountMap[acctName]; ok {
						d["account"] = acctID
					}
					if oid, err := create("contacts", models.AppCRM, d); err == nil {
						if email, ok := d["email"].(string); ok {
							contactMap[email] = oid
						}
					}
				}
				for _, d := range data.CRM.Leads {
					create("leads", models.AppCRM, d)
				}
				logger.Info("Seeded Accounts & Contacts")

				// --- 4. CRM: Opps, Tasks, Meetings, Calls ---
				for _, d := range data.CRM.Opportunities {
					acctName, _ := d["account_name"].(string)
					days, _ := d["close_in_days"].(float64)
					delete(d, "account_name")
					delete(d, "close_in_days")
					if acctID, ok := accountMap[acctName]; ok {
						d["account"] = acctID
					}
					d["close_date"] = time.Now().AddDate(0, 0, int(days))
					create("opportunities", models.AppCRM, d)
				}

				for _, d := range data.CRM.Tasks {
					relName, _ := d["related_to_name"].(string)
					days, _ := d["due_in_days"].(float64)
					assignAdmin, _ := d["assign_to_admin"].(bool)
					delete(d, "related_to_name")
					delete(d, "due_in_days")
					delete(d, "assign_to_admin")

					if acctID, ok := accountMap[relName]; ok {
						d["related_to"] = acctID // Assumption: Polymorphic ID, strict lookup is account
					}
					d["due_date"] = time.Now().AddDate(0, 0, int(days))
					if assignAdmin && userIDStr != "" {
						d["assigned_to"] = userIDStr
					}
					create("tasks", models.AppCRM, d)
				}

				for _, d := range data.CRM.Meetings {
					email, _ := d["contact_email"].(string)
					startH, _ := d["start_in_hours"].(float64)
					durH, _ := d["duration_hours"].(float64)
					delete(d, "contact_email")
					delete(d, "start_in_hours")
					delete(d, "duration_hours")

					if cID, ok := contactMap[email]; ok {
						d["contact"] = cID
					}
					start := time.Now().Add(time.Duration(startH) * time.Hour)
					d["start_time"] = start
					d["end_time"] = start.Add(time.Duration(durH) * time.Hour)
					create("meetings", models.AppCRM, d)
				}

				for _, d := range data.CRM.Calls {
					email, _ := d["contact_email"].(string)
					startH, _ := d["start_in_hours"].(float64)
					delete(d, "contact_email")
					delete(d, "start_in_hours")
					if cID, ok := contactMap[email]; ok {
						d["contact"] = cID
					}
					d["start_time"] = time.Now().Add(time.Duration(startH) * time.Hour)
					create("calls", models.AppCRM, d)
				}
				logger.Info("Seeded CRM Activities")

				// --- 5. Orders ---
				processLines := func(lines []interface{}) []map[string]interface{} {
					newLines := []map[string]interface{}{}
					for _, l := range lines {
						if lm, ok := l.(map[string]interface{}); ok {
							pName, _ := lm["product_name"].(string)
							delete(lm, "product_name")
							if pID, ok := productMap[pName]; ok {
								lm["product"] = pID
							}
							newLines = append(newLines, lm)
						}
					}
					return newLines
				}

				for _, d := range data.Orders.SalesOrders {
					custName, _ := d["customer_name"].(string)
					lines, _ := d["line_items"].([]interface{})
					delete(d, "customer_name")
					if custID, ok := accountMap[custName]; ok {
						d["customer"] = custID
					}
					d["order_date"] = time.Now()
					if lines != nil {
						d["line_items"] = processLines(lines)
					}
					create("sales_orders", models.AppERP, d)
				}

				for _, d := range data.Orders.PurchaseOrders {
					vendName, _ := d["vendor_name"].(string)
					lines, _ := d["line_items"].([]interface{})
					delete(d, "vendor_name")
					if vID, ok := vendorMap[vendName]; ok {
						d["vendor"] = vID
					}
					d["order_date"] = time.Now()
					if lines != nil {
						d["line_items"] = processLines(lines)
					}
					create("purchase_orders", models.AppERP, d)
				}
				logger.Info("Seeded Orders")

				logger.Info("✅ JSON Data Seeding Complete.")
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
			record.NewRecordRepository,
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(SeedTenantData),
	)

	if err := app.Start(context.Background()); err != nil {
	}
	<-app.Done()
}
