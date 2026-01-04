package resource

import (
	"context"

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
// @Description Get resources filtered for sidebar display, filtered by product from X-Rich-Product header
// @Tags Resources
// @Accept json
// @Produce json
// @Param location query string false "Location filter (e.g., main, settings)"
// @Param X-Rich-Product header string true "Product filter (e.g., crm, erp, analytics)"
// @Success 200 {array} Resource "List of sidebar resources grouped by category"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /api/resources/sidebar [get]
func (c *ResourceController) GetSidebarResources(ctx *fiber.Ctx) error {
	product := ctx.Get("X-Rich-Product", "")
	location := ctx.Query("location", "")

	resources, err := c.service.ListSidebarResources(ctx.UserContext(), product, location)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(resources)
}

// ListResources godoc
// @Summary List all resources
// @Description Get all resources for the current tenant, filtered by product via X-Rich-Product header
// @Tags Resources
// @Accept json
// @Produce json
// @Param X-Rich-Product header string true "Product filter (e.g., crm, erp, analytics)"
// @Success 200 {array} Resource "List of all resources"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /api/resources [get]
func (c *ResourceController) ListResources(ctx *fiber.Ctx) error {
	// Get product from header and add to context
	userCtx := ctx.UserContext()
	product := ctx.Get("X-Rich-Product", "")
	if product != "" {
		userCtx = context.WithValue(userCtx, "product", product)
	}

	resources, err := c.service.ListResources(userCtx)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(resources)
}

// GetResourceMetadata godoc
// @Summary Get resource metadata
// @Description Get resource schema and allowed filters based on permissions
// @Tags Resources
// @Accept json
// @Produce json
// @Param resource path string true "Resource ID (e.g., crm.leads)"
// @Param action query string false "Action to check (default: read)"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /me/resources/{resource} [get]
func (c *ResourceController) GetResourceMetadata(ctx *fiber.Ctx) error {
	resourceName := ctx.Params("resource")
	action := ctx.Query("action", "read")

	// Get User ID from Locals (set by AuthMiddleware)
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	metadata, err := c.service.GetResourceMetadata(ctx.UserContext(), resourceName, action, userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(metadata)
}
