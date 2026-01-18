package user

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/core/audit"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService interface {
	ListUsers(ctx context.Context, filter map[string]any, page, limit int64) ([]models.User, int64, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, id string, updates map[string]any) error
	UpdateUserRoles(ctx context.Context, id string, appRoles []models.UserAppRole) error
	UpdateUserStatus(ctx context.Context, id string, status string) error
	DeleteUser(ctx context.Context, id string) error
	InviteUser(ctx context.Context, email string, roleIDs []string, appRoles []models.UserAppRole) (*models.User, error)
	AcceptInvite(ctx context.Context, token, password, firstName, lastName string) error
}

type UserServiceImpl struct {
	UserRepo     UserRepository
	AuditService audit.AuditService
}

func NewUserService(userRepo UserRepository, auditService audit.AuditService) UserService {
	return &UserServiceImpl{
		UserRepo:     userRepo,
		AuditService: auditService,
	}
}

func (s *UserServiceImpl) ListUsers(ctx context.Context, filter map[string]any, page, limit int64) ([]models.User, int64, error) {
	if filter == nil {
		filter = make(map[string]any)
	}

	offset := (page - 1) * limit
	users, total, err := s.UserRepo.List(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserServiceImpl) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.UserRepo.FindByID(ctx, id)
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, user *models.User) error {
	// Initialize default fields if missing
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	if user.Status == "" {
		user.Status = "active"
	}

	// Create in database
	if err := s.UserRepo.Create(ctx, user); err != nil {
		return err
	}

	// Audit Log
	changes := map[string]models.Change{
		"email":   {New: user.Email},
		"created": {New: true},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user", user.ID.Hex(), changes)

	return nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, id string, updates map[string]any) error {
	// Get existing user
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Track changes for audit log
	changes := make(map[string]models.Change)

	// Update fields
	if email, ok := updates["email"].(string); ok && email != user.Email {
		changes["email"] = models.Change{Old: user.Email, New: email}
		user.Email = email
	}
	if firstName, ok := updates["first_name"].(string); ok && firstName != user.FirstName {
		changes["first_name"] = models.Change{Old: user.FirstName, New: firstName}
		user.FirstName = firstName
	}
	if lastName, ok := updates["last_name"].(string); ok && lastName != user.LastName {
		changes["last_name"] = models.Change{Old: user.LastName, New: lastName}
		user.LastName = lastName
	}
	if phone, ok := updates["phone"].(string); ok && phone != user.Phone {
		changes["phone"] = models.Change{Old: user.Phone, New: phone}
		user.Phone = phone
	}
	if status, ok := updates["status"].(string); ok && status != user.Status {
		changes["status"] = models.Change{Old: user.Status, New: status}
		user.Status = status
	}
	if appGroups, ok := updates["app_groups"].([]models.UserAppGroup); ok {
		changes["app_groups"] = models.Change{Old: user.AppGroups, New: appGroups}
		user.AppGroups = appGroups
	}
	if appRoles, ok := updates["app_roles"].([]models.UserAppRole); ok {
		changes["app_roles"] = models.Change{Old: user.AppRoles, New: appRoles}
		user.AppRoles = appRoles
	}

	user.UpdatedAt = time.Now()

	// Update in database
	if err := s.UserRepo.Update(ctx, id, user); err != nil {
		return err
	}

	// Audit log
	if len(changes) > 0 {
		_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, "user", id, changes)
	}

	return nil
}

func (s *UserServiceImpl) UpdateUserRoles(ctx context.Context, id string, appRoles []models.UserAppRole) error {
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Track change
	changes := map[string]models.Change{
		"app_roles": {Old: user.AppRoles, New: appRoles},
	}

	user.AppRoles = appRoles
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Update(ctx, id, user); err != nil {
		return err
	}

	// Audit log
	_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, "user", id, changes)

	return nil
}

func (s *UserServiceImpl) UpdateUserStatus(ctx context.Context, id string, status string) error {
	// Validate status
	validStatuses := []string{"active", "inactive", "suspended"}
	isValid := false
	for _, s := range validStatuses {
		if s == status {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("invalid status: must be active, inactive, or suspended")
	}

	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Track change
	changes := map[string]models.Change{
		"status": {Old: user.Status, New: status},
	}

	user.Status = status
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Update(ctx, id, user); err != nil {
		return err
	}

	// Audit log
	_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, "user", id, changes)

	return nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, id string) error {
	// Delete user
	if err := s.UserRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Audit log
	changes := map[string]models.Change{
		"__deleted": {Old: false, New: true},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionDelete, "user", id, changes)

	return nil
}

func (s *UserServiceImpl) InviteUser(ctx context.Context, email string, roleIDs []string, appRoles []models.UserAppRole) (*models.User, error) {
	// Check if user already exists
	// Note: FindByEmail might need to be scoped to tenant if we allow same email in diff tenants?
	// For now assuming email is unique per tenant or global?
	// The current system seems to assume somewhat global uniqueness or at least tenant context is handled in repo.

	existingUser, err := s.UserRepo.FindByEmail(ctx, email)
	if err == nil {
		if existingUser.Status == "active" {
			return nil, errors.New("user already active")
		}
		// If invited, we can resend invite (regenerate token)
		// For now just error
		return nil, errors.New("user already exists (status: " + existingUser.Status + ")")
	}

	// Generate Token
	token := primitive.NewObjectID().Hex() // Simple token for now
	expiresAt := time.Now().Add(48 * time.Hour)

	// Convert role IDs
	var roles []primitive.ObjectID
	for _, rid := range roleIDs {
		oid, err := primitive.ObjectIDFromHex(rid)
		if err == nil {
			roles = append(roles, oid)
		}
	}

	// Create User
	newUser := &models.User{
		ID:              primitive.NewObjectID(),
		Email:           email,
		Status:          models.StatusInvited,
		AppRoles:        appRoles,
		InviteToken:     token,
		InviteExpiresAt: &expiresAt,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Get TenantID from context
	tenantIDRaw := ctx.Value(models.TenantIDKey)
	if tenantIDStr, ok := tenantIDRaw.(string); ok {
		if tid, err := primitive.ObjectIDFromHex(tenantIDStr); err == nil {
			newUser.TenantID = tid
		}
	}

	if err := s.UserRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	// Audit
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user_invite", newUser.ID.Hex(), nil)

	// TODO: Send Email with Token
	// fmt.Printf("INVITE LINK: /accept-invite?token=%s\n", token)

	return newUser, nil
}

func (s *UserServiceImpl) AcceptInvite(ctx context.Context, token, password, firstName, lastName string) error {
	// Find user by token
	// We need a method for this in Repo.
	// Or we can list users with filter? ListUsers supports filter.

	filter := map[string]any{
		"invite_token": token,
	}
	users, _, err := s.UserRepo.List(ctx, filter, 1, 0)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errors.New("invalid or expired token")
	}

	user := &users[0]

	if user.InviteExpiresAt != nil && time.Now().After(*user.InviteExpiresAt) {
		return errors.New("token expired")
	}

	// Updates
	user.Password = password // TODO: Hash
	user.FirstName = firstName
	user.LastName = lastName
	user.Status = "active"
	user.InviteToken = ""
	user.InviteExpiresAt = nil
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Update(ctx, user.ID.Hex(), user); err != nil {
		return err
	}

	// Audit
	_ = s.AuditService.LogChange(ctx, models.AuditActionUpdate, "user_accept_invite", user.ID.Hex(), nil)

	return nil
}
