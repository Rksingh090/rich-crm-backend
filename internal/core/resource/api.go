package resource

import (
	"github.com/gofiber/fiber/v2"
)

func Setup(router fiber.Router, controller *ResourceController) {
	resources := router.Group("/resources")

	// Public endpoints (require authentication)
	resources.Get("/", controller.GetResources)
	resources.Get("/discover", controller.DiscoverResourcesByType)
	resources.Get("/:resource_id", controller.GetResourceByID)

	// Protected endpoints (require specific permissions)
	resources.Post("/", controller.RegisterResource)
	resources.Put("/:id", controller.UpdateResource)
	resources.Delete("/:id", controller.DeleteResource)
}
