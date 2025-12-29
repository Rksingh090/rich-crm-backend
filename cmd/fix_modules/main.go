package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Initialize Config & DB
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(cfg.DBName)
	mongoDB := &database.MongodbDB{DB: db}

	fmt.Println("🔧 Starting Module Fix...")

	modules := []models.Module{
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
				{Name: "category", Label: "Category", Type: models.FieldTypeSelect, Required: false, Options: []models.SelectOptions{{Label: "Hardware", Value: "Hardware"}, {Label: "Software", Value: "Software"}, {Label: "Service", Value: "Service"}}},
				{Name: "stock", Label: "Stock Quantity", Type: models.FieldTypeNumber, Required: true},
			},
		},
		{
			Name:     "sales_orders", // Keeping inconsistent naming because DB collection module_sales_orders exists
			Label:    "Sales Orders",
			IsSystem: true,
			Fields: []models.ModuleField{
				{Name: "order_number", Label: "Order Number", Type: models.FieldTypeText, Required: true},
				{Name: "date", Label: "Order Date", Type: models.FieldTypeDate, Required: true},
				{Name: "account", Label: "Customer", Type: models.FieldTypeLookup, Required: true, Lookup: &models.LookupDef{LookupModule: "accounts", LookupLabel: "name", ValueField: "_id"}},
				{
					Name: "status", Label: "Status", Type: "select", Required: true,
					Options: []models.SelectOptions{
						{Label: "Draft", Value: "Draft"},
						{Label: "Confirmed", Value: "Confirmed"},
						{Label: "Shipped", Value: "Shipped"},
						{Label: "Cancelled", Value: "Cancelled"},
					},
				},
			},
		},
		{
			Name:     "order_items",
			Label:    "Order Items",
			IsSystem: true,
			Fields: []models.ModuleField{
				{
					Name: "order_id", Label: "Sales Order", Type: "lookup", Required: true,
					Lookup: &models.LookupDef{
						LookupModule: "sales_orders",
						LookupLabel:  "order_number",
						ValueField:   "_id",
					},
				},
				{
					Name: "product_id", Label: "Product", Type: "lookup", Required: true,
					Lookup: &models.LookupDef{
						LookupModule: "products",
						LookupLabel:  "name",
						ValueField:   "_id",
					},
				},
				{Name: "quantity", Label: "Quantity", Type: "number", Required: true},
				{Name: "unit_price", Label: "Unit Price", Type: "currency", Required: true},
			},
		},
		{
			Name:     "tasks",
			Label:    "Tasks",
			IsSystem: true,
			Fields: []models.ModuleField{
				{Name: "subject", Label: "Subject", Type: "text", Required: true},
				{Name: "due_date", Label: "Due Date", Type: "date", Required: true},
				{
					Name: "priority", Label: "Priority", Type: "select", Required: true,
					Options: []models.SelectOptions{
						{Label: "High", Value: "High"},
						{Label: "Normal", Value: "Normal"},
						{Label: "Low", Value: "Low"},
					},
				},
				{
					Name: "status", Label: "Status", Type: "select", Required: true,
					Options: []models.SelectOptions{
						{Label: "Not Started", Value: "Not Started"},
						{Label: "In Progress", Value: "In Progress"},
						{Label: "Completed", Value: "Completed"},
						{Label: "Deferred", Value: "Deferred"},
					},
				},
				{Name: "related_to", Label: "Related To", Type: "text", Required: false},
			},
		},
		{
			Name:     "calls",
			Label:    "Calls",
			IsSystem: true,
			Fields: []models.ModuleField{
				{Name: "subject", Label: "Subject", Type: "text", Required: true},
				{
					Name: "call_type", Label: "Call Type", Type: "select", Required: true,
					Options: []models.SelectOptions{
						{Label: "Inbound", Value: "Inbound"},
						{Label: "Outbound", Value: "Outbound"},
					},
				},
				{Name: "start_time", Label: "Start Time", Type: "datetime", Required: true},
				{Name: "duration", Label: "Duration (min)", Type: "number", Required: true},
				{
					Name: "status", Label: "Status", Type: "select", Required: true,
					Options: []models.SelectOptions{
						{Label: "Scheduled", Value: "Scheduled"},
						{Label: "Completed", Value: "Completed"},
						{Label: "Missed", Value: "Missed"},
					},
				},
				{Name: "related_to", Label: "Related To", Type: "text", Required: false},
			},
		},
		{
			Name:     "meetings",
			Label:    "Meetings",
			IsSystem: true,
			Fields: []models.ModuleField{
				{Name: "subject", Label: "Subject", Type: "text", Required: true},
				{Name: "location", Label: "Location", Type: "text", Required: false},
				{Name: "start_time", Label: "Start Time", Type: "datetime", Required: true},
				{Name: "end_time", Label: "End Time", Type: "datetime", Required: true},
				{Name: "participants", Label: "Participants", Type: "text", Required: false},
				{Name: "related_to", Label: "Related To", Type: "text", Required: false},
			},
		},
		// Legacy 'sales' module from seed/main.go - ensuring it's also updated just in case frontend looks there
		{
			Name:     "sales",
			Label:    "Sales (Legacy)",
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

	moduleCol := mongoDB.DB.Collection("modules")

	for _, mod := range modules {
		// Upsert logic
		filter := bson.M{"name": mod.Name}
		update := bson.M{
			"$set": bson.M{
				"label":      mod.Label,
				"fields":     mod.Fields,
				"is_system":  true,
				"updated_at": time.Now(),
			},
			"$setOnInsert": bson.M{
				"created_at": time.Now(),
			},
		}

		opts := options.Update().SetUpsert(true)
		result, err := moduleCol.UpdateOne(ctx, filter, update, opts)

		if err != nil {
			log.Printf("❌ Failed to update module %s: %v\n", mod.Name, err)
		} else {
			if result.UpsertedCount > 0 {
				fmt.Printf("✅ Created module: %s\n", mod.Name)
			} else {
				fmt.Printf("✅ Updated module: %s\n", mod.Name)
			}
		}
	}

	fmt.Println("🏁 Module Fix Complete!")
}
