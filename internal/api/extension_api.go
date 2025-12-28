package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service" // This line appears to be a duplicate or malformed from the instruction. Keeping it as per instruction.

	"github.com/gofiber/fiber/v2"
)

type ExtensionApi struct {
	Controller  *controllers.ExtensionController
	Config      *config.Config
	RoleService service.RoleService
}

func NewExtensionApi(controller *controllers.ExtensionController, cfg *config.Config, roleService service.RoleService) *ExtensionApi {
	return &ExtensionApi{
		Controller:  controller,
		Config:      cfg,
		RoleService: roleService,
	}
}

func (a *ExtensionApi) Setup(app *fiber.App) {
	api := app.Group("/api/extensions")

	// Apply Auth Middleware
	api.Use(middleware.AuthMiddleware(a.Config.SkipAuth))

	api.Get("/", middleware.RequirePermission(a.RoleService, "marketplace", "read"), a.Controller.ListExtensions)
	api.Get("/:id", middleware.RequirePermission(a.RoleService, "marketplace", "read"), a.Controller.GetExtension)
	api.Post("/:id/install", middleware.RequirePermission(a.RoleService, "marketplace", "update"), a.Controller.InstallExtension)
	api.Post("/:id/uninstall", middleware.RequirePermission(a.RoleService, "marketplace", "update"), a.Controller.UninstallExtension)

	// Admin only / Marketplace Create (Publishing extensions)
	api.Post("/", middleware.RequirePermission(a.RoleService, "marketplace", "create"), a.Controller.CreateExtension)
}
