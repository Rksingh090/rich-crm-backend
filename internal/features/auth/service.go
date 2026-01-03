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

	"fmt"
	"go-crm/internal/features/organization"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthService interface {
	Register(ctx context.Context, username, password, email, orgName string) (*models.User, error)
	Login(ctx context.Context, username, password string) (string, error)
}

type AuthServiceImpl struct {
	UserRepo         user.UserRepository
	RoleRepo         role.RoleRepository
	OrganizationRepo organization.OrganizationRepository
	AuditService     audit.AuditService
}

func NewAuthService(userRepo user.UserRepository, roleRepo role.RoleRepository, orgRepo organization.OrganizationRepository, auditService audit.AuditService) AuthService {
	return &AuthServiceImpl{
		UserRepo:         userRepo,
		RoleRepo:         roleRepo,
		OrganizationRepo: orgRepo,
		AuditService:     auditService,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, username, password, email, orgName string) (*models.User, error) {
	// hash password placeholder (TODO: use bcrypt)
	hashedPassword := password

	// Create Organization
	if orgName == "" {
		orgName = fmt.Sprintf("%s's Organization", username)
	}

	newOrg := models.Organization{
		ID:        primitive.NewObjectID(),
		Name:      orgName,
		Slug:      utils.Slugify(orgName) + "-" + primitive.NewObjectID().Hex()[:4], // Simple slug generation
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Note: OwnerID will be set after user creation, or perform transaction?
	// For simplicity, generate UserID first.
	newUserID := primitive.NewObjectID()
	newOrg.OwnerID = newUserID

	if err := s.OrganizationRepo.Create(ctx, &newOrg); err != nil {
		return nil, err
	}

	// Set Organization Context for subsequent calls (e.g. Roles)
	ctx = context.WithValue(ctx, models.TenantIDKey, newOrg.ID.Hex())

	// Assign default "user" role
	userRole, err := s.RoleRepo.FindByName(ctx, "user")
	var roleIDs []primitive.ObjectID

	switch err {
	case nil:
		roleIDs = append(roleIDs, userRole.ID)
	case mongo.ErrNoDocuments:
		// Create default role if not exists (Bootstrap)
		newRole := role.Role{
			ID:          primitive.NewObjectID(),
			Name:        "user",
			Description: "Default user role",
			Permissions: make(map[string]map[string]models.ActionPermission),
			IsSystem:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.RoleRepo.Create(ctx, &newRole); err == nil {
			roleIDs = append(roleIDs, newRole.ID)
		}
	default:
		return nil, err
	}

	newUser := models.User{
		ID:        newUserID,
		TenantID:  newOrg.ID,
		Username:  username,
		Password:  hashedPassword,
		Email:     email,
		Status:    "active",
		Roles:     roleIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.UserRepo.Create(ctx, &newUser); err != nil {
		// potential rollback of org creation needed here in real world
		return nil, err
	}

	// Audit Log
	changes := map[string]models.Change{
		"username":  {New: username},
		"email":     {New: email},
		"tenant_id": {New: newOrg.ID.Hex()},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user", newUser.ID.Hex(), changes)

	return &newUser, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, username, password string) (string, error) {
	// Use Global lookup because we don't have org context yet
	usr, err := s.UserRepo.FindByUsernameGlobal(ctx, username)
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

	// Set Organization Context for subsequent calls (e.g. Roles)
	ctx = context.WithValue(ctx, models.TenantIDKey, usr.TenantID.Hex())

	// Fetch role names
	var roleNames []string
	var roleIDs []string

	for _, roleID := range usr.Roles {
		r, err := s.RoleRepo.FindByID(ctx, roleID.Hex())
		if err == nil {
			roleNames = append(roleNames, r.Name)
			roleIDs = append(roleIDs, roleID.Hex())
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
	userGroups := usr.Groups
	if userGroups == nil {
		userGroups = []string{}
	}
	token, err := utils.GenerateToken(usr.ID, usr.TenantID, roleNames, roleIDs, userGroups)

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
