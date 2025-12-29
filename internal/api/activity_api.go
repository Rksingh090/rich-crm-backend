package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ActivityApi struct {
	ActivityController *controllers.ActivityController
	RoleService        service.RoleService
	Config             *config.Config
}

func NewActivityApi(activityController *controllers.ActivityController, roleService service.RoleService, config *config.Config) *ActivityApi {
	return &ActivityApi{
		ActivityController: activityController,
		RoleService:        roleService,
		Config:             config,
	}
}

func (api *ActivityApi) Setup(app *fiber.App) {
	group := app.Group("/api/activities", middleware.AuthMiddleware(api.Config.SkipAuth))

	// For now, assume general read permissions or specific 'activities' resource
	// Ideally, we'd check permissions for each underlying module, but for a combined view
	// we might just check if user can read Tasks, Calls, OR Meetings.
	// Simplifying to a basic check for now.
	group.Get("/calendar", api.ActivityController.GetCalendarEvents)
}
