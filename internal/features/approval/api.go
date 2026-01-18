package approval

import (
	"go-crm/internal/config"
	"go-crm/internal/core/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ApprovalApi struct {
	controller  *ApprovalController
	roleService role.RoleService
	config      *config.Config
}

func NewApprovalApi(controller *ApprovalController, roleService role.RoleService, config *config.Config) *ApprovalApi {
	return &ApprovalApi{
		controller:  controller,
		roleService: roleService,
		config:      config,
	}
}

func (h *ApprovalApi) Setup(app *fiber.App) {
	// Group: /workflows
	workflows := app.Group("/api/workflows", middleware.AuthMiddleware(h.config.SkipAuth))

	workflows.Post("/", h.controller.CreateWorkflow)
	workflows.Put("/:id", h.controller.UpdateWorkflow)
	workflows.Delete("/:id", h.controller.DeleteWorkflow)
	workflows.Get("/", h.controller.ListWorkflows)
	workflows.Get("/:id", h.controller.GetWorkflowByID)
	workflows.Get("/module/:moduleId", h.controller.GetWorkflowByModule)

	// Group: /approvals
	approvals := app.Group("/api/approvals", middleware.AuthMiddleware(h.config.SkipAuth))

	// Approval Actions
	approvals.Post("/:module/:id/approve", h.controller.ApproveRecord)
	approvals.Post("/:module/:id/reject", h.controller.RejectRecord)
}
