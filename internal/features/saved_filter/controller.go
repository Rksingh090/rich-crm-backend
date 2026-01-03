package saved_filter

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SavedFilterController struct {
	FilterService SavedFilterService
}

func NewSavedFilterController(filterService SavedFilterService) *SavedFilterController {
	return &SavedFilterController{
		FilterService: filterService,
	}
}

// CreateFilter godoc
// @Summary Create saved filter
// @Description Save a new filter configuration
// @Tags saved_filters
// @Accept json
// @Produce json
// @Param filter body SavedFilter true "Filter Details"
// @Success 201 {object} SavedFilter
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saved-filters [post]
func (c *SavedFilterController) CreateFilter(ctx *fiber.Ctx) error {
	var filter SavedFilter
	if err := ctx.BodyParser(&filter); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)
	filter.UserID = userID

	if err := c.FilterService.CreateFilter(ctx.UserContext(), &filter); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(filter)
}

// GetFilter godoc
// @Summary Get saved filter
// @Description Get a saved filter by ID
// @Tags saved_filters
// @Produce json
// @Param id path string true "Filter ID"
// @Success 200 {object} SavedFilter
// @Failure 404 {object} map[string]interface{}
// @Router /api/saved-filters/{id} [get]
func (c *SavedFilterController) GetFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	filter, err := c.FilterService.GetFilter(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Filter not found"})
	}

	return ctx.JSON(filter)
}

// UpdateFilter godoc
// @Summary Update saved filter
// @Description Update an existing saved filter
// @Tags saved_filters
// @Accept json
// @Produce json
// @Param id path string true "Filter ID"
// @Param filter body SavedFilter true "Filter Details"
// @Success 200 {object} SavedFilter
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saved-filters/{id} [put]
func (c *SavedFilterController) UpdateFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var filter SavedFilter
	if err := ctx.BodyParser(&filter); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	objID, _ := primitive.ObjectIDFromHex(id)
	filter.ID = objID

	if err := c.FilterService.UpdateFilter(ctx.UserContext(), &filter); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(filter)
}

// DeleteFilter godoc
// @Summary Delete saved filter
// @Description Delete a saved filter by ID
// @Tags saved_filters
// @Param id path string true "Filter ID"
// @Success 204 {object} nil
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saved-filters/{id} [delete]
func (c *SavedFilterController) DeleteFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	if err := c.FilterService.DeleteFilter(ctx.UserContext(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// ListUserFilters godoc
// @Summary List user filters
// @Description List stored filters for the current user and module
// @Tags saved_filters
// @Produce json
// @Param module query string true "Module Name"
// @Success 200 {array} SavedFilter
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saved-filters/user [get]
func (c *SavedFilterController) ListUserFilters(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module parameter required"})
	}

	userIDStr, ok := ctx.Locals("user_id").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	filters, err := c.FilterService.GetUserFilters(ctx.UserContext(), userID, moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(filters)
}

// ListPublicFilters godoc
// @Summary List public filters
// @Description List shared/public filters for a module
// @Tags saved_filters
// @Produce json
// @Param module query string true "Module Name"
// @Success 200 {array} SavedFilter
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/saved-filters/public [get]
func (c *SavedFilterController) ListPublicFilters(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module parameter required"})
	}

	filters, err := c.FilterService.GetPublicFilters(ctx.UserContext(), moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(filters)
}
