package resource

import (
	"go-crm/internal/common/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceController struct {
	ResourceService ResourceService
}

func NewResourceController(resourceService ResourceService) *ResourceController {
	return &ResourceController{
		ResourceService: resourceService,
	}
}

// GetResources godoc
// @Summary Get all resources for the current tenant and app
// @Description Retrieves all resources (global, app-level, and tenant-specific) for the authenticated tenant
// @Tags resources
// @Accept json
// @Produce json
// @Param app query string false "App filter (crm, erp, analytics)"
// @Success 200 {array} Resource
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/resources [get]
func (rc *ResourceController) GetResources(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "tenant_id not found in context",
		})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tenant_id",
		})
	}

	app := models.App(c.Query("app", "crm"))

	resources, err := rc.ResourceService.GetResourcesForTenant(c.Context(), tenantID, app)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resources)
}

// GetResourceByID godoc
// @Summary Get a specific resource by ID
// @Description Retrieves a resource by its resource_id
// @Tags resources
// @Accept json
// @Produce json
// @Param resource_id path string true "Resource ID (e.g., crm.leads)"
// @Success 200 {object} Resource
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/resources/{resource_id} [get]
func (rc *ResourceController) GetResourceByID(c *fiber.Ctx) error {
	resourceID := c.Params("resource_id")
	if resourceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "resource_id is required",
		})
	}

	tenantIDStr, ok := c.Locals("tenant_id").(string)
	var tenantID *primitive.ObjectID
	if ok && tenantIDStr != "" {
		tid, err := primitive.ObjectIDFromHex(tenantIDStr)
		if err == nil {
			tenantID = &tid
		}
	}

	resource, err := rc.ResourceService.GetResourceByID(c.Context(), resourceID, tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resource)
}

// RegisterResource godoc
// @Summary Register a new resource
// @Description Creates a new tenant-specific resource (typically called by Module Builder)
// @Tags Resources
// @Accept json
// @Produce json
// @Param resource body Resource true "Resource to register"
// @Success 201 {object} Resource
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/resources [post]
func (rc *ResourceController) RegisterResource(c *fiber.Ctx) error {
	var resource Resource
	if err := c.BodyParser(&resource); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "tenant_id not found in context",
		})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tenant_id",
		})
	}

	resource.TenantID = &tenantID

	if err := rc.ResourceService.RegisterResource(c.Context(), &resource); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resource)
}

// UpdateResource godoc
// @Summary Update a resource
// @Description Updates an existing resource (only non-system resources can be updated)
// @Tags Resources
// @Accept json
// @Produce json
// @Param id path string true "Resource ID"
// @Param resource body Resource true "Updated resource"
// @Success 200 {object} Resource
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/resources/{id} [put]
func (rc *ResourceController) UpdateResource(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	var resource Resource
	if err := c.BodyParser(&resource); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := rc.ResourceService.UpdateResource(c.Context(), id, &resource); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resource)
}

// DeleteResource godoc
// @Summary Delete a resource
// @Description Soft deletes a tenant-specific resource (system resources cannot be deleted)
// @Tags Resources
// @Accept json
// @Produce json
// @Param id path string true "Resource ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/resources/{id} [delete]
func (rc *ResourceController) DeleteResource(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "tenant_id not found in context",
		})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tenant_id",
		})
	}

	if err := rc.ResourceService.DeleteResource(c.Context(), id, tenantID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DiscoverResourcesByType godoc
// @Summary Discover resources by type
// @Description Retrieves all resources of a specific type for the authenticated tenant
// @Tags Resources
// @Accept json
// @Produce json
// @Param app query string false "App filter (crm, erp, analytics)"
// @Param type query string true "Resource type (module, page, setting, etc.)"
// @Success 200 {array} Resource
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/resources/discover [get]
func (rc *ResourceController) DiscoverResourcesByType(c *fiber.Ctx) error {
	tenantIDStr, ok := c.Locals("tenant_id").(string)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "tenant_id not found in context",
		})
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tenant_id",
		})
	}

	app := models.App(c.Query("app", "crm"))
	resourceType := ResourceType(c.Query("type"))

	if resourceType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "type query parameter is required",
		})
	}

	resources, err := rc.ResourceService.DiscoverResourcesByType(c.Context(), tenantID, app, resourceType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resources)
}
