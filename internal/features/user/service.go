package user

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/common/models"
	"go-crm/internal/features/audit"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService interface {
	ListUsers(ctx context.Context, filter map[string]interface{}, page, limit int64) ([]models.User, int64, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateUserRoles(ctx context.Context, id string, roleIDs []string) error
	UpdateUserStatus(ctx context.Context, id string, status string) error
	DeleteUser(ctx context.Context, id string) error
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

func (s *UserServiceImpl) ListUsers(ctx context.Context, filter map[string]interface{}, page, limit int64) ([]models.User, int64, error) {
	if filter == nil {
		filter = make(map[string]interface{})
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
		"username": {New: user.Username},
		"email":    {New: user.Email},
		"created":  {New: true},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user", user.ID.Hex(), changes)

	return nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
	// Get existing user
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Track changes for audit log
	changes := make(map[string]models.Change)

	// Update fields
	if username, ok := updates["username"].(string); ok && username != user.Username {
		changes["username"] = models.Change{Old: user.Username, New: username}
		user.Username = username
	}
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
	if groups, ok := updates["groups"].([]interface{}); ok {
		// Convert []interface{} to []string
		var newGroups []string
		for _, g := range groups {
			if str, ok := g.(string); ok {
				newGroups = append(newGroups, str)
			}
		}
		changes["groups"] = models.Change{Old: user.Groups, New: newGroups}
		user.Groups = newGroups
	}
	if roles, ok := updates["roles"].([]primitive.ObjectID); ok {
		changes["roles"] = models.Change{Old: user.Roles, New: roles}
		user.Roles = roles
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

func (s *UserServiceImpl) UpdateUserRoles(ctx context.Context, id string, roleIDs []string) error {
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Convert string IDs to ObjectIDs
	var objectIDs []primitive.ObjectID
	for _, roleID := range roleIDs {
		oid, err := primitive.ObjectIDFromHex(roleID)
		if err != nil {
			return errors.New("invalid role ID: " + roleID)
		}
		objectIDs = append(objectIDs, oid)
	}

	// Track change
	changes := map[string]models.Change{
		"roles": {Old: user.Roles, New: objectIDs},
	}

	user.Roles = objectIDs
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
	// Get user for audit log
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete user
	if err := s.UserRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Audit log
	changes := map[string]models.Change{
		"deleted":  {Old: false, New: true},
		"username": {Old: user.Username, New: ""},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionDelete, "user", id, changes)

	return nil
}
