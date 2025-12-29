package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type DashboardApi struct {
	DashboardController *controllers.DashboardController
	Config              *config.Config
	RoleService         service.RoleService
}

func NewDashboardApi(dashboardController *controllers.DashboardController, config *config.Config, roleService service.RoleService) Route {
	return &DashboardApi{
		DashboardController: dashboardController,
		Config:              config,
		RoleService:         roleService,
	}
}

func (api *DashboardApi) Setup(app *fiber.App) {
	group := app.Group("/api/dashboards", middleware.AuthMiddleware(api.Config.SkipAuth))

	// Dashboard CRUD
	group.Post("/", api.DashboardController.CreateDashboard)
	group.Get("/", api.DashboardController.ListDashboards)
	group.Get("/:id", api.DashboardController.GetDashboard)
	group.Put("/:id", api.DashboardController.UpdateDashboard)
	group.Delete("/:id", api.DashboardController.DeleteDashboard)

	// Dashboard specific endpoints
	group.Post("/:id/set-default", api.DashboardController.SetDefaultDashboard)
	group.Get("/:id/data", api.DashboardController.GetDashboardData)
}
