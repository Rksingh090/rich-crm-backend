package cron_feature

import (
	"go-crm/internal/config"
	"go-crm/internal/features/role"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type CronApi struct {
	cronController *CronController
	config         *config.Config
	roleService    role.RoleService
}

func NewCronApi(
	cronController *CronController,
	config *config.Config,
	roleService role.RoleService,
) *CronApi {
	return &CronApi{
		cronController: cronController,
		config:         config,
		roleService:    roleService,
	}
}

func (h *CronApi) Setup(app *fiber.App) {
	cronJobs := app.Group("/api/cron-jobs", middleware.AuthMiddleware(h.config.SkipAuth))

	cronJobs.Post("/", middleware.RequirePermission(h.roleService, "automation", "create"), h.cronController.CreateCronJob)
	cronJobs.Get("/", middleware.RequirePermission(h.roleService, "automation", "read"), h.cronController.ListCronJobs)
	cronJobs.Get("/:id", middleware.RequirePermission(h.roleService, "automation", "read"), h.cronController.GetCronJob)
	cronJobs.Put("/:id", middleware.RequirePermission(h.roleService, "automation", "update"), h.cronController.UpdateCronJob)
	cronJobs.Delete("/:id", middleware.RequirePermission(h.roleService, "automation", "delete"), h.cronController.DeleteCronJob)

	cronJobs.Post("/:id/execute", middleware.RequirePermission(h.roleService, "automation", "update"), h.cronController.ExecuteCronJob)
	cronJobs.Get("/:id/logs", middleware.RequirePermission(h.roleService, "automation", "read"), h.cronController.GetCronJobLogs)
}
