package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ReportApi struct {
	ReportController *controllers.ReportController
	Config           *config.Config
	RoleService      service.RoleService
}

func NewReportApi(reportController *controllers.ReportController, config *config.Config, roleService service.RoleService) Route {
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
}
