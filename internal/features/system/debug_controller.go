package system

import (
	"github.com/gofiber/fiber/v2"
)

type DebugController struct{}

func NewDebugController() *DebugController {
	return &DebugController{}
}

// GetCurrentUser godoc
// @Summary      Get current user info
// @Description  Get the current user's info from JWT
// @Tags         debug
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /debug/user [get]
func (c *DebugController) GetCurrentUser(ctx *fiber.Ctx) error {
	userID := ctx.Locals("userID")
	roles := ctx.Locals("roles")

	return ctx.JSON(fiber.Map{
		"user_id": userID,
		"roles":   roles,
		"message": "This is your current JWT token data",
	})
}
