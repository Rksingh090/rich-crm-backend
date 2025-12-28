package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/models"
	"go-crm/internal/repository"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to DB Direct
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	dbWrap := &database.MongodbDB{DB: client.Database(cfg.DBName)}

	userRepo := repository.NewUserRepository(dbWrap)
	roleRepo := repository.NewRoleRepository(dbWrap)

	ctx := context.Background()

	// 3. Find User (assuming 'admin' or list first one)
	// You can change 'admin' to the specific username usually used
	username := "admin"
	if len(os.Args) > 1 {
		username = os.Args[1]
	}

	user, err := userRepo.FindByUsername(ctx, username)
	if err != nil {
		log.Fatalf("Failed to find user %s: %v", username, err)
	}
	fmt.Printf("User: %s (ID: %s)\n", user.Username, user.ID.Hex())
	fmt.Printf("Roles: %v\n", user.Roles)

	// 4. Simulate Logic
	moduleName := "accounts" // The module user complained about
	finalPerms, err := getFieldPermissions(ctx, user, roleRepo, moduleName)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if finalPerms == nil {
		fmt.Println("Result: FULL ACCESS (nil map returned)")
	} else {
		fmt.Printf("Result: Restricted Permissions: %+v\n", finalPerms)
	}
}

// Copy-pasted/Adapted logic from record_service.go
func getFieldPermissions(ctx context.Context, user *models.User, roleRepo repository.RoleRepository, moduleName string) (map[string]string, error) {
	finalPerms := make(map[string]string)
	hasFieldRules := false

	fmt.Printf("[Debug] Checking module '%s'\n", moduleName)

	for _, roleID := range user.Roles {
		role, err := roleRepo.FindByID(ctx, roleID.Hex())
		if err != nil || role == nil {
			fmt.Printf("[Debug] Role ID %s not found\n", roleID.Hex())
			continue
		}

		fmt.Printf("[Debug] Role found: %s\n", role.Name)

		if role.FieldPermissions != nil {
			if modPerms, ok := role.FieldPermissions[moduleName]; ok {
				fmt.Printf("[Debug] Role '%s' has rules: %+v\n", role.Name, modPerms)
				hasFieldRules = true
				for field, p := range modPerms {
					current, exists := finalPerms[field]
					if !exists {
						finalPerms[field] = p
					} else {
						// Union: read_write > read_only > none
						if p == "read_write" {
							finalPerms[field] = "read_write"
						} else if p == "read_only" {
							if current == "none" {
								finalPerms[field] = "read_only"
							}
						}
					}
				}
			} else {
				fmt.Printf("[Debug] Role '%s' has NO rules for module '%s'\n", role.Name, moduleName)
			}
		} else {
			fmt.Printf("[Debug] Role '%s' has nil FieldPermissions\n", role.Name)
		}

		if role.FieldPermissions == nil || role.FieldPermissions[moduleName] == nil {
			fmt.Printf("[Debug] Role '%s' grants FULL ACCESS -> Short-circuiting!\n", role.Name)
			return nil, nil
		}
	}

	if !hasFieldRules {
		fmt.Println("[Debug] No field rules found across any roles")
		return nil, nil
	}

	return finalPerms, nil
}
