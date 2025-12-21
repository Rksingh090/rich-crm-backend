package service

import (
	"context"
	"errors"
	"time"

	"go-crm/internal/models"
	"go-crm/internal/repository"
	"go-crm/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthServiceImpl struct {
	UserRepo     repository.UserRepository
	RoleRepo     repository.RoleRepository
	AuditService AuditService
}

func NewAuthService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, auditService AuditService) *AuthServiceImpl {
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

	if err == nil {
		roleIDs = append(roleIDs, userRole.ID)
	} else if err == mongo.ErrNoDocuments {
		// Create default role if not exists (Bootstrap)
		newRole := models.Role{
			ID:          primitive.NewObjectID(),
			Name:        "user",
			Permissions: []string{"read:own_profile"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.RoleRepo.Create(ctx, &newRole); err == nil {
			roleIDs = append(roleIDs, newRole.ID)
		}
	} else {
		return nil, err
	}

	user := models.User{
		ID:        primitive.NewObjectID(),
		Username:  username,
		Password:  hashedPassword,
		Email:     email,
		Roles:     roleIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.UserRepo.Create(ctx, &user); err != nil {
		return nil, err
	}

	// Audit Log
	changes := map[string]models.Change{
		"username": {New: username},
		"email":    {New: email},
	}
	_ = s.AuditService.LogChange(ctx, models.AuditActionCreate, "user", user.ID.Hex(), changes)

	return &user, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.UserRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Check password (TODO: use bcrypt)
	if user.Password != password {
		return "", errors.New("invalid credentials")
	}

	var roleHexIDs []string
	for _, oid := range user.Roles {
		roleHexIDs = append(roleHexIDs, oid.Hex())
	}

	token, err := utils.GenerateToken(user.ID, roleHexIDs)
	if err != nil {
		return "", err
	}

	return token, nil
}
