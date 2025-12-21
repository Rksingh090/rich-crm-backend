package service

import (
	"context"

	"go-crm/internal/models"
)

type AuthService interface {
	Register(ctx context.Context, username, password, email string) (*models.User, error)
	Login(ctx context.Context, username, password string) (string, error)
}

type RoleService interface {
	CreateRole(ctx context.Context, name string, permissions []string) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	GetPermissionsForRoles(ctx context.Context, roleIDHexes []string) ([]string, error)
}

type ModuleService interface {
	CreateModule(ctx context.Context, module *models.Module) error
	GetModuleByName(ctx context.Context, name string) (*models.Module, error)
	ListModules(ctx context.Context) ([]models.Module, error)
	UpdateModule(ctx context.Context, module *models.Module) error
	DeleteModule(ctx context.Context, name string) error
}

type RecordService interface {
	CreateRecord(ctx context.Context, moduleName string, data map[string]interface{}) (interface{}, error)
	GetRecord(ctx context.Context, moduleName, id string) (map[string]any, error)
	ListRecords(ctx context.Context, moduleName string, filters map[string]any, page, limit int64) ([]map[string]any, error)
	UpdateRecord(ctx context.Context, moduleName, id string, data map[string]interface{}) error
	DeleteRecord(ctx context.Context, moduleName, id string) error
}

type AuditService interface {
	LogChange(ctx context.Context, action models.AuditAction, module string, recordID string, changes map[string]models.Change) error
	ListLogs(ctx context.Context, page, limit int64) ([]models.AuditLog, error)
}
