package sync

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type SyncApi struct {
	controller  *SyncController
	config      *config.Config
	roleService middleware.RoleService
}

func NewSyncApi(controller *SyncController, config *config.Config, roleService middleware.RoleService) api.Route {
	return &SyncApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

// Setup registers all sync routes
func (h *SyncApi) Setup(app *fiber.App) {
	syncGroup := app.Group("/api/sync", middleware.AuthMiddleware(h.config.SkipAuth))

	syncGroup.Post("/settings", middleware.RequirePermission(h.roleService, "db_sync", "create"), h.controller.CreateSyncSetting)
	syncGroup.Get("/settings", middleware.RequirePermission(h.roleService, "db_sync", "read"), h.controller.ListSyncSettings)
	syncGroup.Get("/settings/:id", middleware.RequirePermission(h.roleService, "db_sync", "read"), h.controller.GetSyncSetting)
	syncGroup.Put("/settings/:id", middleware.RequirePermission(h.roleService, "db_sync", "update"), h.controller.UpdateSyncSetting)
	syncGroup.Delete("/settings/:id", middleware.RequirePermission(h.roleService, "db_sync", "delete"), h.controller.DeleteSyncSetting)
	syncGroup.Post("/settings/:id/run", middleware.RequirePermission(h.roleService, "db_sync", "update"), h.controller.RunSync)
	syncGroup.Get("/settings/:id/logs", middleware.RequirePermission(h.roleService, "db_sync", "read"), h.controller.ListSyncLogs)
}
