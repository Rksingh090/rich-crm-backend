package resource

import (
	"go-crm/internal/config"
	"go-crm/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type ResourceApi struct {
	controller  *ResourceController
	config      *config.Config
	roleService middleware.RoleService
}

func NewResourceApi(controller *ResourceController, cfg *config.Config, roleService middleware.RoleService) *ResourceApi {
	return &ResourceApi{
		controller:  controller,
		config:      cfg,
		roleService: roleService,
	}
}

// Setup registers resource routes
func (h *ResourceApi) Setup(app *fiber.App) {
	// Resource routes group with auth middleware
	resources := app.Group("/api/v1/resources", middleware.AuthMiddleware(h.config.SkipAuth))

	// Public endpoints (require authentication)
	resources.Get("/", h.controller.GetResources)
	resources.Get("/discover", h.controller.DiscoverResourcesByType)
	resources.Get("/:resource_id", h.controller.GetResourceByID)

	// Protected endpoints (require specific permissions)
	resources.Post("/", middleware.RequirePermission(h.roleService, "resources", "create"), h.controller.RegisterResource)
	resources.Put("/:id", middleware.RequirePermission(h.roleService, "resources", "update"), h.controller.UpdateResource)
	resources.Delete("/:id", middleware.RequirePermission(h.roleService, "resources", "delete"), h.controller.DeleteResource)
}
