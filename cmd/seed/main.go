package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/module"
	"go-crm/internal/features/organization"
	"go-crm/internal/features/permission"
	"go-crm/internal/features/resource"
	"go-crm/internal/features/role"
	"go-crm/internal/features/user"
	"go-crm/internal/logger"
	"go-crm/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Seed runs the database seeding
func Seed(
	lc fx.Lifecycle,
	roleRepo role.RoleRepository,
	userRepo user.UserRepository,
	moduleRepo module.ModuleRepository,
	orgRepo organization.OrganizationRepository,
	resourceService resource.ResourceService,
	permissionRepo permission.PermissionRepository,
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
				rolesPath := "cmd/seed/data/roles.json"
				usersPath := "cmd/seed/data/users.json"
				modulesPath := "cmd/seed/data/modules.json"
				resourcesPath := "cmd/seed/data/resources.json"
				permissionsPath := "cmd/seed/data/permissions.json"

				// 0. Seed Organization
				orgName := "Default Organization"
				var orgID primitive.ObjectID

				existingOrg, err := orgRepo.FindByName(ctx, orgName)
				if err == nil {
					logger.Info("Organization exists, skipping", zap.String("organization", orgName))
					orgID = existingOrg.ID
				} else {
					// Use a fixed ObjectID for Default Organization for development consistency
					fixedOrgID, _ := primitive.ObjectIDFromHex("678e9a1b2c3d4e5f6a7b8c9e")
					newOrg := common_models.Organization{
						ID:        fixedOrgID,
						Name:      orgName,
						Slug:      utils.Slugify(orgName),
						OwnerID:   primitive.NilObjectID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					if err := orgRepo.Create(ctx, &newOrg); err != nil {
						logger.Fatal("Failed to create organization", zap.Error(err))
					}
					logger.Info("Organization created", zap.String("organization", orgName))
					orgID = newOrg.ID
				}

				// Enforce Organization Context for subsequent repos
				ctx = context.WithValue(ctx, common_models.TenantIDKey, orgID.Hex())

				// 1. Seed Resources
				var resources []resource.Resource
				if err := readJSON(resourcesPath, &resources); err != nil {
					logger.Warn("Failed to read resources.json, skipping resource seeding", zap.Error(err))
				} else {
					if err := resourceService.SyncResources(ctx, resources); err != nil {
						logger.Error("Failed to sync resources", zap.Error(err))
					} else {
						logger.Info("Resources synced successfully", zap.Int("count", len(resources)))
					}
				}

				// 2. Seed Roles
				var roles []role.Role
				if err := readJSON(rolesPath, &roles); err != nil {
					logger.Fatal("Failed to read roles.json", zap.Error(err))
				}

				createdRoles := make(map[string]primitive.ObjectID)

				for _, role := range roles {
					role.TenantID = orgID
					existing, err := roleRepo.FindByName(ctx, role.Name)
					if err == nil {
						logger.Info("Role exists, updating permissions", zap.String("role", role.Name))
						existing.Permissions = role.Permissions
						existing.UpdatedAt = time.Now()
						if err := roleRepo.Update(ctx, existing.ID.Hex(), existing); err != nil {
							logger.Error("Failed to update role", zap.Error(err))
						}
						createdRoles[role.Name] = existing.ID
						continue
					}

					role.ID = primitive.NewObjectID()
					role.CreatedAt = time.Now()
					role.UpdatedAt = time.Now()
					// Role create needs OrgID which is set above

					if err := roleRepo.Create(ctx, &role); err != nil {
						logger.Error("Failed to create role", zap.String("role", role.Name), zap.Error(err))
						continue
					}
					logger.Info("Role created", zap.String("role", role.Name))
					createdRoles[role.Name] = role.ID
				}

				// 2.5. Seed Permissions
				var permissionsData []struct {
					RoleName   string                                    `json:"role_name"`
					Resource   permission.ResourceRef                    `json:"resource"`
					Actions    map[string]common_models.ActionPermission `json:"actions"`
					FieldRules map[string]string                         `json:"field_rules"`
				}
				if err := readJSON(permissionsPath, &permissionsData); err != nil {
					logger.Warn("Failed to read permissions.json, skipping permission seeding", zap.Error(err))
				} else {
					permissionCount := 0
					for _, pData := range permissionsData {
						// Find role ID
						roleID, ok := createdRoles[pData.RoleName]
						if !ok {
							// Try to find in DB
							r, err := roleRepo.FindByName(ctx, pData.RoleName)
							if err != nil {
								logger.Warn("Role not found for permission", zap.String("role", pData.RoleName))
								continue
							}
							roleID = r.ID
						}

						// Check if permission already exists
						existing, err := permissionRepo.FindByRoleAndResource(ctx, roleID.Hex(), pData.Resource.ID)
						if err == nil && existing != nil {
							// Update existing permission
							existing.Actions = pData.Actions
							existing.FieldRules = pData.FieldRules
							existing.UpdatedAt = time.Now()
							if err := permissionRepo.Update(ctx, existing.ID.Hex(), existing); err != nil {
								logger.Error("Failed to update permission", zap.String("role", pData.RoleName), zap.String("resource", pData.Resource.ID), zap.Error(err))
							} else {
								logger.Info("Permission updated", zap.String("role", pData.RoleName), zap.String("resource", pData.Resource.ID))
								permissionCount++
							}
							continue
						}

						// Create new permission
						newPerm := permission.Permission{
							ID:         primitive.NewObjectID(),
							TenantID:   orgID,
							RoleID:     roleID,
							Resource:   pData.Resource,
							Actions:    pData.Actions,
							FieldRules: pData.FieldRules,
							CreatedAt:  time.Now(),
							UpdatedAt:  time.Now(),
						}

						if err := permissionRepo.Create(ctx, &newPerm); err != nil {
							logger.Error("Failed to create permission", zap.String("role", pData.RoleName), zap.String("resource", pData.Resource.ID), zap.Error(err))
						} else {
							logger.Info("Permission created", zap.String("role", pData.RoleName), zap.String("resource", pData.Resource.ID))
							permissionCount++
						}
					}
					logger.Info("Permissions synced successfully", zap.Int("count", permissionCount))
				}

				// 3. Seed Users
				var usersData []struct {
					Username  string   `json:"username"`
					Password  string   `json:"password"`
					Email     string   `json:"email"`
					FirstName string   `json:"first_name"`
					LastName  string   `json:"last_name"`
					Status    string   `json:"status"`
					RoleNames []string `json:"roles"`
					Groups    []string `json:"groups"`
				}
				if err := readJSON(usersPath, &usersData); err != nil {
					logger.Error("Failed to read users.json", zap.Error(err))
				} else {
					for _, uData := range usersData {
						var roleIDs []primitive.ObjectID
						for _, rName := range uData.RoleNames {
							if rID, ok := createdRoles[rName]; ok {
								roleIDs = append(roleIDs, rID)
							} else {
								r, err := roleRepo.FindByName(ctx, rName)
								if err == nil {
									roleIDs = append(roleIDs, r.ID)
								} else {
									logger.Warn("Role found in user definition but not in DB", zap.String("role", rName))
								}
							}
						}

						existingUser, err := userRepo.FindByUsername(ctx, uData.Username)
						if err == nil {
							logger.Info("User exists, updating roles", zap.String("username", uData.Username))
							existingUser.Roles = roleIDs
							existingUser.TenantID = orgID
							existingUser.UpdatedAt = time.Now()
							if err := userRepo.Update(ctx, existingUser.ID.Hex(), existingUser); err != nil {
								logger.Error("Failed to update user roles", zap.String("username", uData.Username), zap.Error(err))
							}
							continue
						}

						newUser := common_models.User{
							ID:        primitive.NewObjectID(),
							Username:  uData.Username,
							Password:  uData.Password,
							Email:     uData.Email,
							FirstName: uData.FirstName,
							LastName:  uData.LastName,
							Status:    uData.Status,
							Roles:     roleIDs,
							Groups:    uData.Groups,
							TenantID:  orgID,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}

						if err := userRepo.Create(ctx, &newUser); err != nil {
							logger.Error("Failed to create user", zap.String("username", uData.Username), zap.Error(err))
						} else {
							logger.Info("User created", zap.String("username", uData.Username))

							if uData.Username == "root" {
								currentOrg, err := orgRepo.FindByID(ctx, orgID.Hex())
								if err == nil && (currentOrg.OwnerID == primitive.NilObjectID || currentOrg.OwnerID.IsZero()) {
									currentOrg.OwnerID = newUser.ID
									if err := orgRepo.Update(ctx, currentOrg); err != nil {
										logger.Error("Failed to update organization owner", zap.Error(err))
									} else {
										logger.Info("Assigned root user as organization owner", zap.String("org", currentOrg.Name), zap.String("user", newUser.Username))
									}
								}
							}
						}
					}
				}

				// 4. Seed Modules (Schema only - Permissions handled by Roles/Resources)
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

				for _, module := range modules {
					// Set Product
					if module.Product == "" {
						if crmModules[module.Name] {
							module.Product = common_models.ProductCRM
						} else if erpModules[module.Name] {
							module.Product = common_models.ProductERP
						} else {
							module.Product = common_models.ProductCRM // Default
						}
					}

					existing, err := moduleRepo.FindByName(ctx, module.Name)
					if err == nil {
						logger.Info("Module exists, checking for field updates", zap.String("module", module.Name))

						// Merge fields
						updated := false
						existingFieldMap := make(map[string]bool)
						for _, f := range existing.Fields {
							existingFieldMap[f.Name] = true
						}

						for _, newField := range module.Fields {
							if !existingFieldMap[newField.Name] {
								existing.Fields = append(existing.Fields, newField)
								logger.Info("Adding new field to module", zap.String("module", module.Name), zap.String("field", newField.Name))
								updated = true
							}
						}

						if existing.Product == "" {
							existing.Product = module.Product
							updated = true
						}

						if updated {
							existing.UpdatedAt = time.Now()
							if err := moduleRepo.Update(ctx, existing); err != nil {
								logger.Error("Failed to update module", zap.String("module", module.Name), zap.Error(err))
							} else {
								logger.Info("Module updated", zap.String("module", module.Name))
							}
						}
						continue
					}

					module.ID = primitive.NewObjectID()
					// TenantID set by repo
					module.CreatedAt = time.Now()
					module.UpdatedAt = time.Now()

					if err := moduleRepo.Create(ctx, &module); err != nil {
						logger.Error("Failed to create module", zap.String("module", module.Name), zap.Error(err))
					} else {
						logger.Info("Module created", zap.String("module", module.Name), zap.String("product", string(module.Product)))
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
			role.NewRoleRepository,
			role.NewRoleService,
			user.NewUserRepository,
			fx.Annotate(
				user.NewUserRepository,
				fx.As(new(audit.UserFinder)),
			),
			module.NewModuleRepository,
			organization.NewOrganizationRepository,
			resource.NewResourceRepository,
			resource.NewResourceService,
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
