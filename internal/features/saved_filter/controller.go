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

func (c *SavedFilterController) CreateFilter(ctx *fiber.Ctx) error {
	var filter SavedFilter
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

func (c *SavedFilterController) GetFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	filter, err := c.FilterService.GetFilter(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Filter not found"})
	}

	return ctx.JSON(filter)
}

func (c *SavedFilterController) UpdateFilter(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var filter SavedFilter
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
