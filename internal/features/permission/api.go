package permission

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type PermissionApi struct {
	Controller *PermissionController
	config     *config.Config
}

func NewPermissionApi(controller *PermissionController, config *config.Config) *PermissionApi {
	return &PermissionApi{
		Controller: controller,
		config:     config,
	}
}

func (a *PermissionApi) Setup(app *fiber.App) {
	api := app.Group("/api")
	RegisterRoutes(api, a.Controller, a.config)
}

// RegisterRoutes registers all permission-related routes
func RegisterRoutes(api fiber.Router, ctrl *PermissionController, config *config.Config) {
	permissions := api.Group("/permissions", middleware.AuthMiddleware(config.SkipAuth))

	permissions.Post("/", ctrl.CreatePermission)
	permissions.Get("/role/:roleId", ctrl.GetPermissionsByRole)
	permissions.Get("/resource", ctrl.GetPermissionsByResource)
	permissions.Put("/:id", ctrl.UpdatePermission)
	permissions.Delete("/:id", ctrl.DeletePermission)
	permissions.Post("/assign", ctrl.AssignResourceToRole)
	permissions.Post("/revoke", ctrl.RevokeResourceFromRole)
	permissions.Get("/effective", ctrl.GetUserEffectivePermissions)
}
