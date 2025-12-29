package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(cfg.DBName)
	mongoDB := &database.MongodbDB{DB: db}

	fmt.Println("🌱 Starting Demo Data Seeding...")

	// 1. Seed Groups
	groups := []models.Group{
		{
			Name:        "Sales Team",
			Description: "Global Sales Team",
			Members:     []primitive.ObjectID{}, // Will populate later if needed
		},
		{
			Name:        "Support Team",
			Description: "Customer Support L1/L2",
			Members:     []primitive.ObjectID{},
		},
		{
			Name:        "Marketing",
			Description: "Marketing & Growth",
			Members:     []primitive.ObjectID{},
		},
	}

	groupCol := mongoDB.DB.Collection("groups")
	var groupIDs []primitive.ObjectID

	for _, g := range groups {
		if count, _ := groupCol.CountDocuments(ctx, bson.M{"name": g.Name}); count == 0 {
			g.ID = primitive.NewObjectID()
			g.CreatedAt = time.Now()
			g.UpdatedAt = time.Now()
			_, err := groupCol.InsertOne(ctx, g)
			if err != nil {
				log.Printf("Failed to create group %s: %v", g.Name, err)
			} else {
				fmt.Printf("Created Group: %s\n", g.Name)
				groupIDs = append(groupIDs, g.ID)
			}
		} else {
			fmt.Printf("Group %s already exists\n", g.Name)
			// Find existing ID
			var existing models.Group
			groupCol.FindOne(ctx, bson.M{"name": g.Name}).Decode(&existing)
			groupIDs = append(groupIDs, existing.ID)
		}
	}

	// 2. Seed/Update Admin Role & Users
	roleCol := mongoDB.DB.Collection("roles")
	var adminRoleID, managerRoleID, userRoleID primitive.ObjectID

	// Force update/create Admin Role to ensure permissions
	adminRoleDef := models.Role{
		Name:        "admin",
		Description: "Administrator with full access",
		IsSystem:    true,
		ModulePermissions: map[string]models.ModulePermission{
			"*": {Create: true, Read: true, Update: true, Delete: true},
		},
		UpdatedAt: time.Now(),
	}

	// Upsert Admin Role
	var existingAdmin models.Role
	err = roleCol.FindOne(ctx, bson.M{"name": "admin"}).Decode(&existingAdmin)
	if err == nil {
		// Update existing
		adminRoleID = existingAdmin.ID
		_, err := roleCol.UpdateOne(ctx, bson.M{"_id": adminRoleID}, bson.M{"$set": bson.M{
			"module_permissions": adminRoleDef.ModulePermissions,
			"updated_at":         time.Now(),
		}})
		if err != nil {
			log.Printf("Failed to update admin role permissions: %v", err)
		} else {
			fmt.Println("Updated Admin Role permissions to full access")
		}
	} else {
		// Create new
		adminRoleDef.ID = primitive.NewObjectID()
		adminRoleDef.CreatedAt = time.Now()
		_, err := roleCol.InsertOne(ctx, adminRoleDef)
		if err != nil {
			log.Printf("Failed to create admin role: %v", err)
		} else {
			fmt.Println("Created Admin Role with full access")
			adminRoleID = adminRoleDef.ID
		}
	}

	// Fetch other roles
	var managerRole models.Role
	err = roleCol.FindOne(ctx, bson.M{"name": "manager"}).Decode(&managerRole)
	if err == nil {
		managerRoleID = managerRole.ID
	}
	var userRole models.Role
	err = roleCol.FindOne(ctx, bson.M{"name": "user"}).Decode(&userRole)
	if err == nil {
		userRoleID = userRole.ID
	}

	users := []models.User{
		{
			Username:  "admin_demo",
			Password:  "admin@123",
			Email:     "admin.demo@example.com",
			FirstName: "Admin",
			LastName:  "Demo",
			Status:    "active",
			Roles:     []primitive.ObjectID{adminRoleID},
		},
		{
			Username:  "demo_manager",
			Password:  "password123",
			Email:     "manager@demo.com",
			FirstName: "Demo",
			LastName:  "Manager",
			Status:    "active",
			Roles:     []primitive.ObjectID{managerRoleID},
		},
		{
			Username:  "demo_sales",
			Password:  "password123",
			Email:     "sales@demo.com",
			FirstName: "John",
			LastName:  "Sales",
			Status:    "active",
			Roles:     []primitive.ObjectID{userRoleID},
		},
		{
			Username:  "demo_support",
			Password:  "password123",
			Email:     "support@demo.com",
			FirstName: "Jane",
			LastName:  "Support",
			Status:    "active",
			Roles:     []primitive.ObjectID{userRoleID},
		},
	}

	userCol := mongoDB.DB.Collection("users")
	for _, u := range users {
		if count, _ := userCol.CountDocuments(ctx, bson.M{"username": u.Username}); count == 0 {
			u.ID = primitive.NewObjectID()
			u.CreatedAt = time.Now()
			u.UpdatedAt = time.Now()
			_, err := userCol.InsertOne(ctx, u)
			if err != nil {
				log.Printf("Failed to create user %s: %v", u.Username, err)
			} else {
				fmt.Printf("Created User: %s (Password: %s)\n", u.Username, u.Password)
			}
		} else {
			fmt.Printf("User %s already exists\n", u.Username)
		}
	}

	// 3. Seed Accounts (Module: accounts)
	accountCol := mongoDB.DB.Collection("module_accounts")
	accountNames := []string{"Acme Corp", "Globex", "Soylent Corp", "Initech", "Umbrella Corp", "Massive Dynamic", "Hooli", "Pied Piper", "Stark Industries", "Wayne Enterprises"}
	var accountIDs []primitive.ObjectID

	for _, name := range accountNames {
		if count, _ := accountCol.CountDocuments(ctx, bson.M{"name": name}); count == 0 {
			doc := bson.M{
				"_id":        primitive.NewObjectID(),
				"name":       name,
				"industry":   randomChoice([]string{"Tech", "Finance", "Healthcare", "Retail"}),
				"website":    fmt.Sprintf("https://www.%s.com", toSlug(name)),
				"phone":      fmt.Sprintf("555-01%2d", rand.Intn(99)),
				"type":       randomChoice([]string{"Customer", "Partner", "Vendor"}),
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}
			_, err := accountCol.InsertOne(ctx, doc)
			if err == nil {
				fmt.Printf("Created Account: %s\n", name)
				accountIDs = append(accountIDs, doc["_id"].(primitive.ObjectID))
			}
		} else {
			// Get ID
			var existing bson.M
			accountCol.FindOne(ctx, bson.M{"name": name}).Decode(&existing)
			accountIDs = append(accountIDs, existing["_id"].(primitive.ObjectID))
		}
	}

	// 4. Seed Contacts (Module: contacts)
	contactCol := mongoDB.DB.Collection("module_contacts")
	firstNames := []string{"Alice", "Bob", "Charlie", "David", "Eve", "Frank", "Grace", "Heidi", "Ivan", "Judy"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez"}

	for i := 0; i < 20; i++ {
		first := randomChoice(firstNames)
		last := randomChoice(lastNames)
		email := fmt.Sprintf("%s.%s@example.com", toSlug(first), toSlug(last)) // Simplistic uniqueness

		// Ensure unique email by appending random chars if needed, but for demo simplistic is fine or check
		if count, _ := contactCol.CountDocuments(ctx, bson.M{"email": email}); count == 0 {
			// Assign to random account
			accountID := accountIDs[rand.Intn(len(accountIDs))]

			doc := bson.M{
				"_id":        primitive.NewObjectID(),
				"first_name": first,
				"last_name":  last,
				"email":      email,
				"phone":      "555-123-4567",
				"account":    accountID,
				"title":      randomChoice([]string{"CEO", "CTO", "Manager", "developer"}),
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}
			contactCol.InsertOne(ctx, doc)
			fmt.Printf("Created Contact: %s %s\n", first, last)
		}
	}

	// 5. Seed Leads (Module: leads)
	leadCol := mongoDB.DB.Collection("module_leads")
	for i := 0; i < 20; i++ {
		first := randomChoice(firstNames)
		last := randomChoice(lastNames)
		name := fmt.Sprintf("%s %s", first, last)
		email := fmt.Sprintf("lead.%s.%d@example.com", toSlug(first), rand.Intn(1000))

		doc := bson.M{
			"_id":        primitive.NewObjectID(),
			"name":       name,
			"email":      email,
			"phone":      "555-987-6543",
			"company":    fmt.Sprintf("%s Inc", last),
			"status":     randomChoice([]string{"New", "Contacted", "Qualified", "Lost"}),
			"source":     randomChoice([]string{"Web", "Referral", "Event"}),
			"created_at": time.Now(),
			"updated_at": time.Now(),
		}
		leadCol.InsertOne(ctx, doc)
		fmt.Printf("Created Lead: %s\n", name)
	}

	// 6. Seed Products (Module: products)
	productCol := mongoDB.DB.Collection("module_products")
	products := []struct {
		Name  string
		Price float64
	}{
		{"Laptop Pro", 1200}, {"Smartphone X", 800}, {"Tablet Mini", 400}, {"Headphones", 150}, {"Monitor 4K", 500},
		{"Mouse", 50}, {"Keyboard", 100}, {"Dock Station", 200}, {"Webcam", 80}, {"Microphone", 120},
	}

	var productIDs []primitive.ObjectID
	for _, p := range products {
		if count, _ := productCol.CountDocuments(ctx, bson.M{"name": p.Name}); count == 0 {
			doc := bson.M{
				"_id":         primitive.NewObjectID(),
				"name":        p.Name,
				"code":        fmt.Sprintf("PRD-%d", rand.Intn(10000)),
				"price":       p.Price,
				"description": fmt.Sprintf("High quality %s", p.Name),
				"category":    randomChoice([]string{"Hardware", "Electronics"}),
				"stock":       rand.Intn(100) + 10, // Added stock field as per seed_modules
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			}
			_, err := productCol.InsertOne(ctx, doc)
			if err == nil {
				fmt.Printf("Created Product: %s\n", p.Name)
				productIDs = append(productIDs, doc["_id"].(primitive.ObjectID))
			}
		} else {
			var existing bson.M
			productCol.FindOne(ctx, bson.M{"name": p.Name}).Decode(&existing)
			productIDs = append(productIDs, existing["_id"].(primitive.ObjectID))
		}
	}

	// 7. Seed SLA Policies
	slaCol := mongoDB.DB.Collection("sla_policies")
	// Policies
	slaPolicies := []models.SLAPolicy{
		{
			Name:           "Standard SLA",
			Description:    "Standard support for all users",
			Priority:       models.TicketPriorityMedium,
			ResponseTime:   120, // 2 hours
			ResolutionTime: 480, // 8 hours
			IsActive:       true,
		},
		{
			Name:           "Urgent SLA",
			Description:    "For critical issues",
			Priority:       models.TicketPriorityUrgent,
			ResponseTime:   30,  // 30 mins
			ResolutionTime: 240, // 4 hours
			IsActive:       true,
		},
	}

	for _, sla := range slaPolicies {
		if count, _ := slaCol.CountDocuments(ctx, bson.M{"name": sla.Name}); count == 0 {
			sla.ID = primitive.NewObjectID()
			sla.CreatedAt = time.Now()
			sla.UpdatedAt = time.Now()
			slaCol.InsertOne(ctx, sla)
			fmt.Printf("Created SLA: %s\n", sla.Name)
		}
	}

	// 8. Fetch Helpers for Relational Data
	cursor, _ := accountCol.Find(ctx, bson.M{})
	var accounts []bson.M
	cursor.All(ctx, &accounts)
	var allAccountIDs []primitive.ObjectID
	for _, a := range accounts {
		allAccountIDs = append(allAccountIDs, a["_id"].(primitive.ObjectID))
	}

	// 9. Seed Opportunities (linked to Accounts)
	if len(allAccountIDs) > 0 {
		oppCol := mongoDB.DB.Collection("module_opportunities")
		stages := []string{"Prospecting", "Negotiation", "Closed Won", "Closed Lost"}
		for i := 0; i < 15; i++ {
			accID := allAccountIDs[rand.Intn(len(allAccountIDs))]
			name := fmt.Sprintf("Opp #%d", rand.Intn(1000))
			if count, _ := oppCol.CountDocuments(ctx, bson.M{"name": name}); count == 0 {
				doc := bson.M{
					"_id":        primitive.NewObjectID(),
					"name":       name,
					"amount":     rand.Float64() * 10000,
					"stage":      randomChoice(stages),
					"close_date": time.Now().AddDate(0, rand.Intn(3), 0),
					"account":    accID,
					"created_at": time.Now(),
					"updated_at": time.Now(),
				}
				oppCol.InsertOne(ctx, doc)
				fmt.Printf("Created Opportunity: %s\n", name)
			}
		}
	}

	// 10. Seed Sales Orders (linked to Accounts)
	if len(allAccountIDs) > 0 {
		salesCol := mongoDB.DB.Collection("module_sales_orders")
		statuses := []string{"Draft", "Confirmed", "Shipped", "Cancelled"}
		for i := 0; i < 10; i++ {
			accID := allAccountIDs[rand.Intn(len(allAccountIDs))]
			orderNum := fmt.Sprintf("SO-%d", rand.Intn(10000))
			if count, _ := salesCol.CountDocuments(ctx, bson.M{"order_number": orderNum}); count == 0 {
				doc := bson.M{
					"_id":          primitive.NewObjectID(),
					"order_number": orderNum,
					"date":         time.Now(),
					"status":       randomChoice(statuses),
					"customer":     accID,
					"account":      accID,
					"created_at":   time.Now(),
					"updated_at":   time.Now(),
				}
				salesCol.InsertOne(ctx, doc)
				fmt.Printf("Created Sales Order: %s\n", orderNum)
			}
		}
	}

	// 11. Seed Tasks
	taskCol := mongoDB.DB.Collection("module_tasks")
	for i := 0; i < 10; i++ {
		subject := fmt.Sprintf("Task %d", i)
		if count, _ := taskCol.CountDocuments(ctx, bson.M{"subject": subject}); count == 0 {
			doc := bson.M{
				"_id":        primitive.NewObjectID(),
				"subject":    subject,
				"due_date":   time.Now().AddDate(0, 0, rand.Intn(7)),
				"priority":   randomChoice([]string{"High", "Normal", "Low"}),
				"status":     randomChoice([]string{"Not Started", "In Progress", "Completed"}),
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}
			taskCol.InsertOne(ctx, doc)
			fmt.Printf("Created Task: %s\n", subject)
		}
	}

	// 12. Seed Calls
	callCol := mongoDB.DB.Collection("module_calls")
	for i := 0; i < 10; i++ {
		subject := fmt.Sprintf("Call %d", i)
		if count, _ := callCol.CountDocuments(ctx, bson.M{"subject": subject}); count == 0 {
			doc := bson.M{
				"_id":        primitive.NewObjectID(),
				"subject":    subject,
				"call_type":  randomChoice([]string{"Inbound", "Outbound"}),
				"start_time": time.Now(),
				"duration":   rand.Intn(60),
				"status":     randomChoice([]string{"Scheduled", "Completed", "Missed"}),
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}
			callCol.InsertOne(ctx, doc)
			fmt.Printf("Created Call: %s\n", subject)
		}
	}

	// 13. Seed Meetings
	meetingCol := mongoDB.DB.Collection("module_meetings")
	for i := 0; i < 5; i++ {
		subject := fmt.Sprintf("Meeting %d", i)
		if count, _ := meetingCol.CountDocuments(ctx, bson.M{"subject": subject}); count == 0 {
			doc := bson.M{
				"_id":          primitive.NewObjectID(),
				"subject":      subject,
				"location":     "Conference Room A",
				"start_time":   time.Now().Add(time.Hour),
				"end_time":     time.Now().Add(2 * time.Hour),
				"participants": "All Stakeholders",
				"created_at":   time.Now(),
				"updated_at":   time.Now(),
			}
			meetingCol.InsertOne(ctx, doc)
			fmt.Printf("Created Meeting: %s\n", subject)
		}
	}

	// 14. Seed Webhooks
	webhookCol := mongoDB.DB.Collection("webhooks")
	webhooks := []models.Webhook{
		{
			URL:         "https://hook.example.com/crm-events",
			Secret:      "demo-secret-key",
			Events:      []string{"record.created", "record.updated"},
			ModuleName:  "leads",
			IsActive:    true,
			Description: "Log all lead changes to external system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, wh := range webhooks {
		if count, _ := webhookCol.CountDocuments(ctx, bson.M{"url": wh.URL}); count == 0 {
			wh.ID = primitive.NewObjectID()
			webhookCol.InsertOne(ctx, wh)
			fmt.Printf("Created Webhook: %s\n", wh.URL)
		}
	}

	// 15. Seed Cron Jobs
	cronCol := mongoDB.DB.Collection("cron_jobs")
	cronJobs := []models.CronJob{
		{
			Name:        "Daily Cleanup",
			Description: "Archive old logs",
			Schedule:    "0 0 * * *", // Daily at midnight
			Active:      true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "Sync External Data",
			Description: "Sync data from external API",
			Schedule:    "0 */4 * * *", // Every 4 hours
			Active:      true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, job := range cronJobs {
		if count, _ := cronCol.CountDocuments(ctx, bson.M{"name": job.Name}); count == 0 {
			job.ID = primitive.NewObjectID()
			cronCol.InsertOne(ctx, job)
			fmt.Printf("Created Cron Job: %s\n", job.Name)
		}
	}

	fmt.Println("✅ Demo Data Seeding Complete!")
}

func randomChoice(options []string) string {
	return options[rand.Intn(len(options))]
}

func toSlug(s string) string {
	// simplistic slug
	return s
}
