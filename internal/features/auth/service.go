package auth

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/role"
	"go-crm/internal/features/user"
	"go-crm/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthService interface {
	Register(ctx context.Context, username, password, email string) (*models.User, error)
	Login(ctx context.Context, username, password string) (string, error)
}

type AuthServiceImpl struct {
	UserRepo     user.UserRepository
	RoleRepo     role.RoleRepository
	AuditService audit.AuditService
}

func NewAuthService(userRepo user.UserRepository, roleRepo role.RoleRepository, auditService audit.AuditService) AuthService {
	return &AuthServiceImpl{
		UserRepo:     userRepo,
		RoleRepo:     roleRepo,
		AuditService: auditService,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, username, password, email string) (*models.User, error) {
	// hash password placeholder (TODO: use bcrypt)
	hashedPassword := password

	// Assign default "user" role
	userRole, err := s.RoleRepo.FindByName(ctx, "user")
	var roleIDs []primitive.ObjectID

	switch err {
	case nil:
		roleIDs = append(roleIDs, userRole.ID)
	case mongo.ErrNoDocuments:
		// Create default role if not exists (Bootstrap)
		newRole := role.Role{
			ID:                primitive.NewObjectID(),
			Name:              "user",
			Description:       "Default user role",
			ModulePermissions: make(map[string]role.ModulePermission),
			IsSystem:          false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		if err := s.RoleRepo.Create(ctx, &newRole); err == nil {
			roleIDs = append(roleIDs, newRole.ID)
		}
	default:
		return nil, err
	}

	newUser := models.User{
		ID:        primitive.NewObjectID(),
		Username:  username,
		Password:  hashedPassword,
		Email:     email,
		Status:    "active",
		Roles:     roleIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.UserRepo.Create(ctx, &newUser); err != nil {
		return nil, err
	}

	// Audit Log
	changes := map[string]models.Change{
		"username": {New: username},
		"email":    {New: email},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user", newUser.ID.Hex(), changes)

	return &newUser, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, username, password string) (string, error) {
	usr, err := s.UserRepo.FindByUsername(ctx, username)
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

	// Fetch role names and aggregate permissions
	var roleNames []string
	var roleIDs []string
	permissions := make(map[string][]string)

	for _, roleID := range usr.Roles {
		r, err := s.RoleRepo.FindByID(ctx, roleID.Hex())
		if err == nil {
			roleNames = append(roleNames, r.Name)
			roleIDs = append(roleIDs, roleID.Hex())

			// Aggregate permissions from all roles
			for moduleName, modulePerm := range r.ModulePermissions {
				if _, exists := permissions[moduleName]; !exists {
					permissions[moduleName] = []string{}
				}

				// Add permissions if granted by this role
				if modulePerm.Create && !contains(permissions[moduleName], "create") {
					permissions[moduleName] = append(permissions[moduleName], "create")
				}
				if modulePerm.Read && !contains(permissions[moduleName], "read") {
					permissions[moduleName] = append(permissions[moduleName], "read")
				}
				if modulePerm.Update && !contains(permissions[moduleName], "update") {
					permissions[moduleName] = append(permissions[moduleName], "update")
				}
				if modulePerm.Delete && !contains(permissions[moduleName], "delete") {
					permissions[moduleName] = append(permissions[moduleName], "delete")
				}
			}
		}
	}

	// If user has admin role, grant full wildcard permission
	isAdmin := false
	for _, name := range roleNames {
		if name == "admin" || name == "Super Admin" {
			isAdmin = true
			break
		}
	}

	if isAdmin {
		permissions["*"] = []string{"create", "read", "update", "delete"}
	}

	// If no roles found, assign empty array
	if roleNames == nil {
		roleNames = []string{}
	}
	if roleIDs == nil {
		roleIDs = []string{}
	}

	token, err := utils.GenerateToken(usr.ID, roleNames, roleIDs, permissions)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
