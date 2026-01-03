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
	// Public resource routes (with auth middleware only)
	resources := app.Group("/api/resources", middleware.AuthMiddleware(h.config.SkipAuth))

	// GET /api/resources/sidebar - Get sidebar resources (public for authenticated users)
	resources.Get("/sidebar", h.controller.GetSidebarResources)

	// GET /api/resources - List all resources (admin only)
	resources.Get("/", middleware.RequirePermission(h.roleService, "resources", "read"), h.controller.ListResources)
}
