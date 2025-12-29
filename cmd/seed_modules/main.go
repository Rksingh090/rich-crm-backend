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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Initialize Config & DB
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(cfg.DBName)
	mongoDB := &database.MongodbDB{DB: db}

	// Define Modules
	modules := []models.Module{
		{
			Name:  "products",
			Label: "Products",
			Fields: []models.ModuleField{
				{Name: "name", Label: "Product Name", Type: "text", Required: true},
				{Name: "sku", Label: "SKU", Type: "text", Required: true},
				{Name: "stock", Label: "Stock Quantity", Type: "number", Required: true},
				{Name: "price", Label: "Price", Type: "currency", Required: true},
			},
		},
		{
			Name:  "sales_orders",
			Label: "Sales Orders",
			Fields: []models.ModuleField{
				{Name: "order_number", Label: "Order Number", Type: "text", Required: true},
				{Name: "date", Label: "Order Date", Type: "date", Required: true},
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
			Name:  "order_items",
			Label: "Order Items",
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
			Name:  "tasks",
			Label: "Tasks",
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
			Name:  "calls",
			Label: "Calls",
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
			Name:  "meetings",
			Label: "Meetings",
			Fields: []models.ModuleField{
				{Name: "subject", Label: "Subject", Type: "text", Required: true},
				{Name: "location", Label: "Location", Type: "text", Required: false},
				{Name: "start_time", Label: "Start Time", Type: "datetime", Required: true},
				{Name: "end_time", Label: "End Time", Type: "datetime", Required: true},
				{Name: "participants", Label: "Participants", Type: "text", Required: false},
				{Name: "related_to", Label: "Related To", Type: "text", Required: false},
			},
		},
	}

	col := mongoDB.DB.Collection("modules")

	for _, mod := range modules {
		// Check if exists
		count, _ := col.CountDocuments(ctx, bson.M{"name": mod.Name})
		if count > 0 {
			fmt.Printf("Module %s already exists. Skipping.\n", mod.Name)
			continue
		}

		mod.ID = primitive.NewObjectID()
		mod.CreatedAt = time.Now()
		mod.UpdatedAt = time.Now()
		mod.IsSystem = true // Mark as system so user cannot delete easily

		_, err := col.InsertOne(ctx, mod)
		if err != nil {
			log.Printf("Failed to create module %s: %v\n", mod.Name, err)
		} else {
			fmt.Printf("Created module: %s\n", mod.Name)
		}
	}

	// Create Automation Rule for Stock Deduction
	// We do this to save user time
	autoCol := mongoDB.DB.Collection("automation_rules")
	ruleName := "Deduct Stock on Ship"
	count, _ := autoCol.CountDocuments(ctx, bson.M{"name": ruleName})
	if count == 0 {
		rule := models.AutomationRule{
			ID:          primitive.NewObjectID(),
			Name:        ruleName,
			ModuleID:    "sales_orders",
			TriggerType: "update",
			Active:      true,
			Conditions: []models.RuleCondition{
				{
					Field:    "status",
					Operator: "equals",
					Value:    "Shipped",
				},
			},
			Actions: []models.RuleAction{
				{
					Type: "run_script",
					Config: map[string]interface{}{
						"script_content": `
order := modules.get("sales_orders", record_id)
// We need to fetch items for this order.
// Note: RecordRepo.List uses a filter map.
items := modules.list("order_items", {"order_id": record_id})

for item in items {
 	// item is an ImmutableMap, need to access fields
 	product_id := item.product_id
	qty := item.quantity

	// Fetch product
	product := modules.get("products", product_id)
	
	// Deduct stock
	// product.stock is likely a float or int.
	// Tengo handles numbers proactively.
	current_stock := product.stock
	new_stock := current_stock - qty
	
	// Update product
	// We pass the entire product object back with updated stock
	product.stock = new_stock
	
	res := modules.update("products", product_id, product)
	if !res {
		log("Failed to update stock for product", product_id)
	} else {
		log("Deducted stock for product", product_id, "New Stock:", new_stock)
	}
}
`,
					},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err := autoCol.InsertOne(ctx, rule)
		if err != nil {
			log.Printf("Failed to create automation rule: %v\n", err)
		} else {
			fmt.Println("Created automation rule: Deduct Stock on Ship")
		}
	}
}
