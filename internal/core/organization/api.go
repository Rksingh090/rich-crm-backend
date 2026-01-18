package organization

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type OrganizationApi struct {
	controller  *OrganizationController
	config      *config.Config
	roleService middleware.RoleService
}

func NewOrganizationApi(controller *OrganizationController, config *config.Config, roleService middleware.RoleService) *OrganizationApi {
	return &OrganizationApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

func (h *OrganizationApi) Setup(app *fiber.App) {
	group := app.Group("/api/organizations", middleware.AuthMiddleware(h.config.SkipAuth))

	// List
	group.Get("/", h.controller.ListOrganizations)
	// Create
	group.Post("/", middleware.RequirePermission(h.roleService, "crm.settings_tenants", "create"), h.controller.CreateOrganization)
	// Get
	group.Get("/:id", h.controller.GetOrganization)
	// Update
	group.Put("/:id", middleware.RequirePermission(h.roleService, "crm.settings_tenants", "edit"), h.controller.UpdateOrganization)
	// Delete
	group.Delete("/:id", middleware.RequirePermission(h.roleService, "crm.settings_tenants", "delete"), h.controller.DeleteOrganization)

	// Users & Activity (Requires view permission on tenants)
	group.Get("/:id/users", middleware.RequirePermission(h.roleService, "crm.settings_tenants", "read"), h.controller.GetOrganizationUsers)
	group.Get("/:id/activity", middleware.RequirePermission(h.roleService, "crm.settings_tenants", "read"), h.controller.GetOrganizationActivity)
}
