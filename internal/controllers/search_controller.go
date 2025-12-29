package controllers

import (
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SearchController struct {
	Service service.SearchService
}

func NewSearchController(service service.SearchService) *SearchController {
	return &SearchController{
		Service: service,
	}
}

// GlobalSearch godoc
// @Summary      Global Search
// @Description  Search across modules, pages, and records
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        q   query string true "Search Query"
// @Success      200 {array} service.SearchResult
// @Router       /api/search [get]
func (ctrl *SearchController) GlobalSearch(c *fiber.Ctx) error {
	query := c.Query("q")

	// Get user ID from context
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	results, err := ctrl.Service.GlobalSearch(c.Context(), query, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(results)
}
