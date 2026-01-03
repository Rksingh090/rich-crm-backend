package report

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ReportApi struct {
	ReportController *ReportController
	Config           *config.Config
	RoleService      middleware.RoleService
}

func NewReportApi(reportController *ReportController, config *config.Config, roleService middleware.RoleService) *ReportApi {
	return &ReportApi{
		ReportController: reportController,
		Config:           config,
		RoleService:      roleService,
	}
}

func (api *ReportApi) Setup(app *fiber.App) {
	group := app.Group("/api/reports", middleware.AuthMiddleware(api.Config.SkipAuth))

	group.Post("/", middleware.RequirePermission(api.RoleService, "reports", "create"), api.ReportController.Create)
	group.Get("/", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.List)
	group.Get("/:id", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.Get)
	group.Put("/:id", middleware.RequirePermission(api.RoleService, "reports", "update"), api.ReportController.Update)
	group.Delete("/:id", middleware.RequirePermission(api.RoleService, "reports", "delete"), api.ReportController.Delete)
	group.Get("/:id/run", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.Run)
	group.Get("/:id/export", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.Export)

	// Advanced reporting endpoints
	group.Post("/pivot", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.RunPivot)
	group.Post("/cross-module", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.RunCrossModule)
	group.Post("/export-excel", middleware.RequirePermission(api.RoleService, "reports", "read"), api.ReportController.ExportExcel)
}
