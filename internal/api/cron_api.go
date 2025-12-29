package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CronApi struct {
	cronController *controllers.CronController
	config         *config.Config
	roleService    service.RoleService
}

func NewCronApi(
	cronController *controllers.CronController,
	config *config.Config,
	roleService service.RoleService,
) *CronApi {
	return &CronApi{
		cronController: cronController,
		config:         config,
		roleService:    roleService,
	}
}

// Setup registers all cron job related routes
func (h *CronApi) Setup(app *fiber.App) {
	// Cron jobs routes group with auth middleware
	cronJobs := app.Group("/api/cron-jobs", middleware.AuthMiddleware(h.config.SkipAuth))

	// CRUD operations - Protected by "automation" permission
	cronJobs.Post("/", middleware.RequirePermission(h.roleService, "automation", "create"), h.cronController.CreateCronJob)
	cronJobs.Get("/", middleware.RequirePermission(h.roleService, "automation", "read"), h.cronController.ListCronJobs)
	cronJobs.Get("/:id", middleware.RequirePermission(h.roleService, "automation", "read"), h.cronController.GetCronJob)
	cronJobs.Put("/:id", middleware.RequirePermission(h.roleService, "automation", "update"), h.cronController.UpdateCronJob)
	cronJobs.Delete("/:id", middleware.RequirePermission(h.roleService, "automation", "delete"), h.cronController.DeleteCronJob)

	// Execution and logs
	cronJobs.Post("/:id/execute", middleware.RequirePermission(h.roleService, "automation", "update"), h.cronController.ExecuteCronJob)
	cronJobs.Get("/:id/logs", middleware.RequirePermission(h.roleService, "automation", "read"), h.cronController.GetCronJobLogs)
}
