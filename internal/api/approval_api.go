package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ApprovalApi struct {
	controller  *controllers.ApprovalController
	roleService service.RoleService
	config      *config.Config
}

func NewApprovalApi(controller *controllers.ApprovalController, roleService service.RoleService, config *config.Config) *ApprovalApi {
	return &ApprovalApi{
		controller:  controller,
		roleService: roleService,
		config:      config,
	}
}

func (h *ApprovalApi) Setup(app *fiber.App) {
	// Group: /workflows
	workflows := app.Group("/api/workflows", middleware.AuthMiddleware(h.config.SkipAuth))

	// Manage Workflows (Admin only or specific permission?)
	// For now, let's say "modules:update" or similar, or new "workflows" module permission
	// Let's stick to "roles:create" level (admin-ish) or create new system module "workflows"
	// Assuming "workflows" permissions will be added to role management later.
	// For now, protect with "workflows" module (needs to be added to seeds/system roles)
	// Or just allow authenticated users for now if we haven't seeded "workflows" permission.

	// Let's REQUIRE "workflows" permission. User will need to add this module permission to their role.
	workflows.Post("/", middleware.RequirePermission(h.roleService, "workflows", "create"), h.controller.CreateWorkflow)
	workflows.Put("/:id", middleware.RequirePermission(h.roleService, "workflows", "update"), h.controller.UpdateWorkflow)
	workflows.Delete("/:id", middleware.RequirePermission(h.roleService, "workflows", "delete"), h.controller.DeleteWorkflow)
	workflows.Get("/", middleware.RequirePermission(h.roleService, "workflows", "read"), h.controller.ListWorkflows)
	workflows.Get("/:id", middleware.RequirePermission(h.roleService, "workflows", "read"), h.controller.GetWorkflowByID)
	workflows.Get("/module/:moduleId", middleware.RequirePermission(h.roleService, "workflows", "read"), h.controller.GetWorkflowByModule)

	// Group: /approvals
	approvals := app.Group("/api/approvals", middleware.AuthMiddleware(h.config.SkipAuth))

	// Approval Actions
	// Actions are protected by logic inside service (CanApprove), so just AuthMiddleware is base requirement
	approvals.Post("/:module/:id/approve", h.controller.ApproveRecord)
	approvals.Post("/:module/:id/reject", h.controller.RejectRecord)
}
