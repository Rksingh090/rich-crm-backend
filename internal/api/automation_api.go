package api

import (
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AutomationApi struct {
	Controller  *controllers.AutomationController
	RoleService service.RoleService
}

func NewAutomationApi(controller *controllers.AutomationController, roleService service.RoleService) *AutomationApi {
	return &AutomationApi{
		Controller:  controller,
		RoleService: roleService,
	}
}

func (a *AutomationApi) Setup(app *fiber.App) {
	group := app.Group("/api/automation")

	// Apply Auth Middleware
	// Assuming automation management requires strict permissions
	group.Use(middleware.AuthMiddleware(false))

	// Permissions could be "automation:read", "automation:write"
	// Using "automation" as module name
	group.Post("/", middleware.RequirePermission(a.RoleService, "automation", "create"), a.Controller.CreateRule)
	group.Get("/", middleware.RequirePermission(a.RoleService, "automation", "read"), a.Controller.ListRules)
	group.Get("/:id", middleware.RequirePermission(a.RoleService, "automation", "read"), a.Controller.GetRule)
	group.Put("/:id", middleware.RequirePermission(a.RoleService, "automation", "update"), a.Controller.UpdateRule)
	group.Delete("/:id", middleware.RequirePermission(a.RoleService, "automation", "delete"), a.Controller.DeleteRule)
}
