package resource

import (
	"github.com/gofiber/fiber/v2"
)

type ResourceController struct {
	service ResourceService
}

func NewResourceController(service ResourceService) *ResourceController {
	return &ResourceController{
		service: service,
	}
}

// GetSidebarResources godoc
// @Summary Get sidebar resources
// @Description Get resources filtered for sidebar display, optionally filtered by product (crm, erp, analytics)
// @Tags Resources
// @Accept json
// @Produce json
// @Param product query string false "Product filter (e.g., crm, erp, analytics)" default(crm)
// @Success 200 {array} Resource "List of sidebar resources grouped by category"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /api/resources/sidebar [get]
func (c *ResourceController) GetSidebarResources(ctx *fiber.Ctx) error {
	product := ctx.Query("product", "")
	location := ctx.Query("location", "")

	resources, err := c.service.ListSidebarResources(ctx.UserContext(), product, location)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(resources)
}

// ListResources godoc
// @Summary List all resources
// @Description Get all resources for the current tenant (admin only)
// @Tags Resources
// @Accept json
// @Produce json
// @Success 200 {array} Resource "List of all resources"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /api/resources [get]
func (c *ResourceController) ListResources(ctx *fiber.Ctx) error {
	resources, err := c.service.ListResources(ctx.UserContext())
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(resources)
}
