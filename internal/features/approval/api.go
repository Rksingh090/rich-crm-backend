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
	// Group: /approval-processes
	processes := app.Group("/api/approval-processes", middleware.AuthMiddleware(h.config.SkipAuth))

	processes.Post("/", h.controller.CreateApprovalProcess)
	processes.Put("/:id", h.controller.UpdateApprovalProcess)
	processes.Delete("/:id", h.controller.DeleteApprovalProcess)
	processes.Get("/", h.controller.ListApprovalProcesses)
	processes.Get("/:id", h.controller.GetApprovalProcessByID)
	processes.Get("/module/:moduleId", h.controller.GetApprovalProcessByModule)

	// Group: /approvals
	approvals := app.Group("/api/approvals", middleware.AuthMiddleware(h.config.SkipAuth))

	// Approval Actions
	approvals.Post("/:module/:id/approve", h.controller.ApproveRecord)
	approvals.Post("/:module/:id/reject", h.controller.RejectRecord)
}
