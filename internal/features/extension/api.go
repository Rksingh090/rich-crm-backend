package extension

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ExtensionApi struct {
	Controller  *ExtensionController
	Config      *config.Config
	RoleService middleware.RoleService
}

func NewExtensionApi(controller *ExtensionController, cfg *config.Config, roleService middleware.RoleService) api.Route {
	return &ExtensionApi{
		Controller:  controller,
		Config:      cfg,
		RoleService: roleService,
	}
}

func (a *ExtensionApi) Setup(app *fiber.App) {
	group := app.Group("/api/extensions")

	group.Use(middleware.AuthMiddleware(a.Config.SkipAuth))

	group.Get("/", middleware.RequirePermission(a.RoleService, "marketplace", "read"), a.Controller.ListExtensions)
	group.Get("/:id", middleware.RequirePermission(a.RoleService, "marketplace", "read"), a.Controller.GetExtension)
	group.Post("/:id/install", middleware.RequirePermission(a.RoleService, "marketplace", "update"), a.Controller.InstallExtension)
	group.Post("/:id/uninstall", middleware.RequirePermission(a.RoleService, "marketplace", "update"), a.Controller.UninstallExtension)

	group.Post("/", middleware.RequirePermission(a.RoleService, "marketplace", "create"), a.Controller.CreateExtension)
}
