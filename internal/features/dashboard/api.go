package dashboard

import (
	"go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type DashboardApi struct {
	DashboardController *DashboardController
	Config              *config.Config
	RoleService         middleware.RoleService
}

func NewDashboardApi(dashboardController *DashboardController, cfg *config.Config, roleService middleware.RoleService) api.Route {
	return &DashboardApi{
		DashboardController: dashboardController,
		Config:              cfg,
		RoleService:         roleService,
	}
}

func (api *DashboardApi) Setup(app *fiber.App) {
	group := app.Group("/api/dashboards", middleware.AuthMiddleware(api.Config.SkipAuth))

	group.Post("/", api.DashboardController.CreateDashboard)
	group.Get("/", api.DashboardController.ListDashboards)
	group.Get("/:id", api.DashboardController.GetDashboard)
	group.Put("/:id", api.DashboardController.UpdateDashboard)
	group.Delete("/:id", api.DashboardController.DeleteDashboard)

	group.Post("/:id/set-default", api.DashboardController.SetDefaultDashboard)
	group.Get("/:id/data", api.DashboardController.GetDashboardData)
}
