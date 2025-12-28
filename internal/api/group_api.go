package api

import (
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type GroupApi struct {
	controller *controllers.GroupController
	config     *config.Config
}

func NewGroupApi(controller *controllers.GroupController, config *config.Config) *GroupApi {
	return &GroupApi{
		controller: controller,
		config:     config,
	}
}

func (h *GroupApi) Setup(app *fiber.App) {
	groups := app.Group("/api/groups", middleware.AuthMiddleware(h.config.SkipAuth))

	groups.Post("/", h.controller.CreateGroup)
	groups.Get("/", h.controller.GetAllGroups)
	groups.Get("/:id", h.controller.GetGroup)
	groups.Put("/:id", h.controller.UpdateGroup)
	groups.Delete("/:id", h.controller.DeleteGroup)

	// Member management
	groups.Post("/:id/members", h.controller.AddMember)
	groups.Delete("/:id/members/:userId", h.controller.RemoveMember)
}
