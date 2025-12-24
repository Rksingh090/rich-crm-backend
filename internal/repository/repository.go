package repository

import (
	"context"

	"go-crm/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context, filter map[string]interface{}, limit, offset int64) ([]models.User, int64, error)
	Update(ctx context.Context, id string, user *models.User) error
	Delete(ctx context.Context, id string) error
}

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	FindByName(ctx context.Context, name string) (*models.Role, error)
	FindPermissionsByRoleIDs(ctx context.Context, roleIDs []interface{}) ([]string, error)
}

type ModuleRepository interface {
	Create(ctx context.Context, module *models.Module) error
	FindByName(ctx context.Context, name string) (*models.Module, error)
	List(ctx context.Context) ([]models.Module, error)
	Update(ctx context.Context, module *models.Module) error
	Delete(ctx context.Context, name string) error
	DropCollection(ctx context.Context, name string) error
	FindUsingLookup(ctx context.Context, targetModule string) ([]models.Module, error)
}

type RecordRepository interface {
	Create(ctx context.Context, moduleName string, data map[string]any) (any, error)
	Get(ctx context.Context, moduleName, id string) (map[string]any, error)
	List(ctx context.Context, moduleName string, filter map[string]any, limit, offset int64) ([]map[string]any, error)
	Update(ctx context.Context, moduleName, id string, data map[string]any) error
	Delete(ctx context.Context, moduleName, id string) error
}

type FileRepository interface {
	Save(ctx context.Context, file *models.File) error
	Get(ctx context.Context, id string) (*models.File, error)
}
