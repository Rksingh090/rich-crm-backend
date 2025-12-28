package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SyncApi struct {
	controller  *controllers.SyncController
	config      *config.Config
	roleService service.RoleService
}

func NewSyncApi(controller *controllers.SyncController, config *config.Config, roleService service.RoleService) *SyncApi {
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
	syncGroup.Post("/settings/:id/run", middleware.RequirePermission(h.roleService, "db_sync", "update"), h.controller.RunSync) // Run = Update (Execute)
	syncGroup.Get("/settings/:id/logs", middleware.RequirePermission(h.roleService, "db_sync", "read"), h.controller.ListSyncLogs)
}
