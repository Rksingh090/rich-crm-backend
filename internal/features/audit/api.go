package audit

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type AuditApi struct {
	controller  *AuditController
	config      *config.Config
	roleService middleware.RoleService
}

func NewAuditApi(controller *AuditController, config *config.Config, roleService middleware.RoleService) *AuditApi {
	return &AuditApi{
		controller:  controller,
		config:      config,
		roleService: roleService,
	}
}

func (h *AuditApi) Setup(app *fiber.App) {
	audit := app.Group("/api/audit-logs", middleware.AuthMiddleware(h.config.SkipAuth))

	audit.Get("/", middleware.RequirePermission(h.roleService, "crm.settings_audit_logs", "read"), h.controller.ListLogs)
}
