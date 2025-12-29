package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SavedFilterController struct {
	FilterService service.SavedFilterService
}

func NewSavedFilterController(filterService service.SavedFilterService) *SavedFilterController {
	return &SavedFilterController{
		FilterService: filterService,
	}
}

// CreateFilter godoc
// @Summary Create saved filter
// @Tags Saved Filters
// @Accept json
// @Produce json
// @Param filter body models.SavedFilter true "Saved Filter"
// @Success 201 {object} models.SavedFilter
// @Router /api/filters [post]
func (c *SavedFilterController) CreateFilter(ctx *fiber.Ctx) error {
	var filter models.SavedFilter
	if err := ctx.BodyParser(&filter); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)
	filter.UserID = userID

	if err := c.FilterService.CreateFilter(ctx.Context(), &filter); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(filter)
}

// GetFilter godoc
// @Summary Get saved filter
// @Tags Saved Filters
// @Produce json
// @Param id path string true "Filter ID"
// @Success 200 {object} models.SavedFilter
// @Router /api/filters/{id} [get]
func (c *SavedFilterController) GetFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	filter, err := c.FilterService.GetFilter(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Filter not found"})
	}

	return ctx.JSON(filter)
}

// UpdateFilter godoc
// @Summary Update saved filter
// @Tags Saved Filters
// @Accept json
// @Produce json
// @Param id path string true "Filter ID"
// @Param filter body models.SavedFilter true "Saved Filter"
// @Success 200 {object} models.SavedFilter
// @Router /api/filters/{id} [put]
func (c *SavedFilterController) UpdateFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var filter models.SavedFilter
	if err := ctx.BodyParser(&filter); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	objID, _ := primitive.ObjectIDFromHex(id)
	filter.ID = objID

	if err := c.FilterService.UpdateFilter(ctx.Context(), &filter); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(filter)
}

// DeleteFilter godoc
// @Summary Delete saved filter
// @Tags Saved Filters
// @Param id path string true "Filter ID"
// @Success 204
// @Router /api/filters/{id} [delete]
func (c *SavedFilterController) DeleteFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	if err := c.FilterService.DeleteFilter(ctx.Context(), id, userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// ListUserFilters godoc
// @Summary List user's saved filters
// @Tags Saved Filters
// @Produce json
// @Param module query string true "Module name"
// @Success 200 {array} models.SavedFilter
// @Router /api/filters [get]
func (c *SavedFilterController) ListUserFilters(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module parameter required"})
	}

	userIDStr, ok := ctx.Locals("userID").(string)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	filters, err := c.FilterService.GetUserFilters(ctx.Context(), userID, moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(filters)
}

// ListPublicFilters godoc
// @Summary List public filters
// @Tags Saved Filters
// @Produce json
// @Param module query string true "Module name"
// @Success 200 {array} models.SavedFilter
// @Router /api/filters/public [get]
func (c *SavedFilterController) ListPublicFilters(ctx *fiber.Ctx) error {
	moduleName := ctx.Query("module")
	if moduleName == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "module parameter required"})
	}

	filters, err := c.FilterService.GetPublicFilters(ctx.Context(), moduleName)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(filters)
}
