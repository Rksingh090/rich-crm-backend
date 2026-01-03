package search

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SearchController struct {
	Service SearchService
}

func NewSearchController(service SearchService) *SearchController {
	return &SearchController{
		Service: service,
	}
}

// GlobalSearch godoc
// GlobalSearch godoc
// @Summary Global search
// @Description Search across all modules
// @Tags search
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/search [get]
func (ctrl *SearchController) GlobalSearch(c *fiber.Ctx) error {
	query := c.Query("q")

	userIDStr, ok := c.Locals("user_id").(string)
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

	results, err := ctrl.Service.GlobalSearch(c.UserContext(), query, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(results)
}
