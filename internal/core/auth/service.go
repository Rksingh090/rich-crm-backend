package auth

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/organization"
	"go-crm/internal/core/permission"
	"go-crm/internal/core/role"
	"go-crm/internal/core/user"
	"go-crm/internal/features/module"
	"go-crm/internal/features/resource"
	"go-crm/pkg/utils"

	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthService interface {
	Register(ctx context.Context, password, email, orgName, plan string, apps []string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	LoginControlPlane(ctx context.Context, email, password string) (string, error)
	CreateTenantWithAdmin(ctx context.Context, org *models.Organization, adminEmail, adminPassword, adminFirstName, adminLastName, roleName string) (*models.User, error)
}

type AuthServiceImpl struct {
	UserRepo         user.UserRepository
	RoleRepo         role.RoleRepository
	ModuleRepo       module.ModuleRepository
	OrganizationRepo organization.OrganizationRepository
	ResourceRepo     resource.ResourceRepository
	PermissionRepo   permission.PermissionRepository
	AuditService     audit.AuditService
}

func NewAuthService(
	userRepo user.UserRepository,
	roleRepo role.RoleRepository,
	moduleRepo module.ModuleRepository,
	orgRepo organization.OrganizationRepository,
	resourceRepo resource.ResourceRepository,
	permissionRepo permission.PermissionRepository,
	auditService audit.AuditService,
) AuthService {
	return &AuthServiceImpl{
		UserRepo:         userRepo,
		RoleRepo:         roleRepo,
		ModuleRepo:       moduleRepo,
		OrganizationRepo: orgRepo,
		ResourceRepo:     resourceRepo,
		PermissionRepo:   permissionRepo,
		AuditService:     auditService,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, password, email, orgName, plan string, apps []string) (*models.User, error) {
	// hash password placeholder (TODO: use bcrypt)
	hashedPassword := password

	// Create Organization
	if orgName == "" {
		// Use email part before @ as default org name
		// Simple implementation
		for i, char := range email {
			if char == '@' {
				orgName = fmt.Sprintf("%s's Organization", email[:i])
				break
			}
		}
		if orgName == "" {
			orgName = "Default Organization"
		}
	}

	var enabledApps []models.App
	for _, app := range apps {
		enabledApps = append(enabledApps, models.App(app))
	}

	newOrg := models.Organization{
		ID:          primitive.NewObjectID(),
		Name:        orgName,
		Slug:        utils.Slugify(orgName) + "-" + primitive.NewObjectID().Hex()[:4], // Simple slug generation
		Plan:        plan,
		CreatedBy:   primitive.NilObjectID, // Placeholder until user created
		EnabledApps: enabledApps,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Note: OwnerID will be set after user creation, or perform transaction?
	// For simplicity, generate UserID first.
	newUserID := primitive.NewObjectID()
	newOrg.OwnerID = newUserID

	if err := s.OrganizationRepo.Create(ctx, &newOrg); err != nil {
		return nil, err
	}

	// 2. Seed Tenant (Copy Defaults)
	adminRoleID, err := s.seedTenant(ctx, newOrg.ID)
	if err != nil {
		return nil, err
	}

	// 3. Set Organization Context for subsequent calls
	ctx = context.WithValue(ctx, models.TenantIDKey, newOrg.ID.Hex())

	// Prepare App Roles
	var userAppRoles []models.UserAppRole
	for _, p := range enabledApps {
		userAppRoles = append(userAppRoles, models.UserAppRole{
			AppID:  p,
			RoleID: adminRoleID,
		})
	}

	newUser := models.User{
		ID:        newUserID,
		TenantID:  newOrg.ID,
		Password:  hashedPassword,
		Email:     email,
		Status:    "active",
		AppRoles:  userAppRoles,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.UserRepo.Create(ctx, &newUser); err != nil {
		return nil, err
	}

	// Audit Log
	changes := map[string]models.Change{
		"email":     {New: email},
		"tenant_id": {New: newOrg.ID.Hex()},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user", newUser.ID.Hex(), changes)

	return &newUser, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, email, password string) (string, error) {
	// Use Global lookup because we don't have org context yet
	usr, err := s.UserRepo.FindByEmailGlobal(ctx, email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Check password (TODO: use bcrypt)
	if usr.Password != password {
		return "", errors.New("invalid credentials")
	}

	// Check user status
	if usr.Status == "suspended" {
		return "", errors.New("account suspended")
	}
	if usr.Status == "inactive" {
		return "", errors.New("account inactive")
	}
	if usr.Status == models.StatusInvited {
		return "", errors.New("account invited, please complete signup")
	}

	// Set Organization Context for subsequent calls (e.g. Roles)
	ctx = context.WithValue(ctx, models.TenantIDKey, usr.TenantID.Hex())

	// Fetch role names
	var roleNames []string
	var roleIDs []string

	// Populate global role lists from App Roles for backward compatibility in JWT
	for _, appRole := range usr.AppRoles {
		r, err := s.RoleRepo.FindByID(ctx, appRole.RoleID.Hex())
		if err == nil {
			roleNames = append(roleNames, r.Name)
			roleIDs = append(roleIDs, appRole.RoleID.Hex())
		}
	}

	// If no roles found, assign empty array
	if roleNames == nil {
		roleNames = []string{}
	}
	if roleIDs == nil {
		roleIDs = []string{}
	}

	// Generate JWT with user groups
	var userGroupStrings []string
	if usr.Groups != nil {
		for _, oid := range usr.Groups {
			userGroupStrings = append(userGroupStrings, oid.Hex())
		}
	}

	// Build App Claims
	var appClaims []utils.AppClaim
	var enabledApps map[string]bool = make(map[string]bool)

	// Fetch Organization to get Enabled Apps
	if usr.TenantID != primitive.NilObjectID {
		org, err := s.OrganizationRepo.FindByID(ctx, usr.TenantID.Hex())
		if err == nil {
			for _, p := range org.EnabledApps {
				enabledApps[string(p)] = true
			}
		}
	}

	// Iterate User App Roles
	for _, appRole := range usr.AppRoles {
		appCode := string(appRole.AppID)
		if enabledApps[appCode] {
			// Fetch Role Name for this App Role
			r, err := s.RoleRepo.FindByID(ctx, appRole.RoleID.Hex())
			roleName := "viewer" // Default fallback
			if err == nil {
				roleName = r.Name
			}

			appClaims = append(appClaims, utils.AppClaim{
				Name: appCode,
				Role: roleName,
			})
		}
	}

	if appClaims == nil {
		appClaims = []utils.AppClaim{}
	}

	token, err := utils.GenerateToken(usr.ID, usr.TenantID, roleNames, roleIDs, userGroupStrings, appClaims, usr.IsPlatformAdmin)

	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthServiceImpl) LoginControlPlane(ctx context.Context, email, password string) (string, error) {
	// 1. Find User by Email Global
	usr, err := s.UserRepo.FindByEmailGlobal(ctx, email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// 2. Check Password
	if usr.Password != password {
		return "", errors.New("invalid credentials")
	}

	// 3. Verify Platform Admin Status
	if !usr.IsPlatformAdmin {
		return "", errors.New("unauthorized access")
	}

	// 4. Check Status
	if usr.Status != "active" {
		return "", errors.New("account is not active")
	}

	// 5. Generate Token (with nil TenantID and IsPlatformAdmin=true)
	// We don't need tenant-specific checks here because this is for the control plane.
	// Platform admins might have a tenant (if they are also a user), but for CP login we want a CP token?
	// OR we just give them a token that reflects who they are.
	// Current GenerateToken takes TenantID. For CP, we might want to pass nil/empty if the session is not bound to a tenant.
	// However, the user model HAS a TenantID.
	// If we want a "Tenant-less" or "All-Tenant" token, maybe pass NilObjectID or handle in claims.
	// For now, let's pass their Home Tenant ID if exists, but IsPlatformAdmin flag is key.

	// Use empty/nil tenant for CP context?
	// If they are managing ANY tenant, having a specific tenant ID in token might restrict them in middleware if middleware checks tenant.
	// But `GenerateToken` signature requires TenantID.
	// Let's use the user's tenant ID but rely on IsPlatformAdmin for permissions.
	// Actually, if they are "Control Plane" users, they might NOT belong to a tenant?
	// The request says "users who are not belongs to tenant".
	// So usr.TenantID might be Nil.

	token, err := utils.GenerateToken(usr.ID, usr.TenantID, []string{"Super Admin"}, nil, nil, nil, true)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthServiceImpl) CreateTenantWithAdmin(ctx context.Context, org *models.Organization, adminEmail, adminPassword, adminFirstName, adminLastName, roleName string) (*models.User, error) {
	// 1. Create Organization
	if org.ID.IsZero() {
		org.ID = primitive.NewObjectID()
	}
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now()
	}
	org.UpdatedAt = time.Now()

	// Pre-generate User ID to set as OwnerID
	newUserID := primitive.NewObjectID()
	org.OwnerID = newUserID

	if err := s.OrganizationRepo.Create(ctx, org); err != nil {
		return nil, err
	}

	// 2. Seed Tenant (Copy Defaults)
	adminRoleID, err := s.seedTenant(ctx, org.ID)
	if err != nil {
		return nil, err
	}

	// 3. Find requested role ID if provided
	targetRoleID := adminRoleID // Default to seeded admin role
	tenantCtx := context.WithValue(ctx, models.TenantIDKey, org.ID.Hex())

	if roleName != "" {
		r, err := s.RoleRepo.FindByName(tenantCtx, roleName)
		if err == nil {
			targetRoleID = r.ID
		}
	}

	// 4. Create User
	// hash password placeholder (TODO: use bcrypt)
	hashedPassword := adminPassword

	// Prepare App Roles
	var userAppRoles []models.UserAppRole
	for _, p := range org.EnabledApps {
		userAppRoles = append(userAppRoles, models.UserAppRole{
			AppID:  p,
			RoleID: targetRoleID,
		})
	}

	newUser := models.User{
		ID:        newUserID,
		TenantID:  org.ID,
		Email:     adminEmail,
		Password:  hashedPassword,
		FirstName: adminFirstName,
		LastName:  adminLastName,
		Status:    "active",
		AppRoles:  userAppRoles,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.UserRepo.Create(tenantCtx, &newUser); err != nil {
		return nil, err
	}

	// Audit
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "tenant_with_admin", org.ID.Hex(), nil)

	return &newUser, nil
}

func (s *AuthServiceImpl) seedTenant(ctx context.Context, tenantID primitive.ObjectID) (primitive.ObjectID, error) {
	tenantCtx := context.WithValue(ctx, models.TenantIDKey, tenantID.Hex())
	var adminRoleID primitive.ObjectID
	roleMap := make(map[string]primitive.ObjectID) // Name -> NewID

	fmt.Printf("Seeding Tenant: %s\n", tenantID.Hex())

	// 1. Copy Roles
	defaultRoles, err := s.RoleRepo.GetDefaults(ctx)
	if err == nil && len(defaultRoles) > 0 {
		for _, r := range defaultRoles {
			oldID := r.ID
			// Skip seeding Super Admin role to tenants
			if r.Name == "Super Admin" {
				continue
			}

			r.ID = primitive.NewObjectID()
			r.TenantID = tenantID
			if r.Name == "admin" {
				adminRoleID = r.ID
			}
			roleMap[r.Name] = r.ID
			_ = s.RoleRepo.Create(tenantCtx, &r)
			fmt.Printf("Seeded Role: %s (Old ID: %s, New ID: %s)\n", r.Name, oldID.Hex(), r.ID.Hex())
		}
	}

	// 2. Copy Modules
	defaultModules, err := s.ModuleRepo.GetDefaults(ctx)
	if err == nil && len(defaultModules) > 0 {
		for _, m := range defaultModules {
			m.ID = primitive.NewObjectID()
			m.TenantID = tenantID
			m.Scope = "tenant"
			_ = s.ModuleRepo.Create(tenantCtx, &m)
			fmt.Printf("Seeded Module: %s\n", m.Name)
		}
	}

	// 3. Copy Resources
	defaultResources, err := s.ResourceRepo.GetDefaults(ctx)
	if err == nil && len(defaultResources) > 0 {
		for _, res := range defaultResources {
			res.ID = primitive.NewObjectID()
			res.TenantID = tenantID
			res.Scope = "tenant"
			if err := s.ResourceRepo.Create(tenantCtx, &res); err != nil {
				fmt.Printf("Error seeding resource %s: %v\n", res.ResourceID, err)
			} else {
				fmt.Printf("Seeded Resource: %s\n", res.ResourceID)
			}
		}
	} else if err != nil {
		fmt.Printf("Error getting default resources: %v\n", err)
	}

	// 4. Copy Permissions (Map to new Role IDs)
	defaultPermissions, err := s.PermissionRepo.GetDefaults(ctx)
	if err == nil && len(defaultPermissions) > 0 {
		for _, p := range defaultPermissions {
			// Find new RoleID from mapping using RoleName
			if newRoleID, ok := roleMap[p.RoleName]; ok {
				p.ID = primitive.NewObjectID()
				p.TenantID = tenantID
				p.RoleID = newRoleID
				_ = s.PermissionRepo.Create(tenantCtx, &p)
				fmt.Printf("Seeded Permission for Role: %s, Resource: %s\n", p.RoleName, p.ResourceID)
			}
		}
	}

	// 5. Ensure Indexes for Tenant
	_ = s.RoleRepo.EnsureIndexes(tenantCtx)
	_ = s.ModuleRepo.EnsureIndexes(tenantCtx)
	_ = s.ResourceRepo.EnsureIndexes(tenantCtx)
	_ = s.PermissionRepo.EnsureIndexes(tenantCtx)

	// Fallback/Bootstrap Admin Role if not seeded from defaults
	if adminRoleID.IsZero() {
		adminRole := &role.Role{
			ID:          primitive.NewObjectID(),
			Name:        "admin",
			Description: "Administrator",
			IsSystem:    true,
			TenantID:    tenantID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Permissions: map[string]map[string]models.ActionPermission{
				"*": {"*": {Allowed: true}},
			},
		}
		if err := s.RoleRepo.Create(tenantCtx, adminRole); err != nil {
			return primitive.NilObjectID, err
		}
		adminRoleID = adminRole.ID
	}

	return adminRoleID, nil
}
