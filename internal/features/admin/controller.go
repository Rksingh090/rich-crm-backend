package admin

import (
	"github.com/gofiber/fiber/v2"
)

// AdminController
type AdminController struct {
}

// NewAdminController returns the pointer to the struct
func NewAdminController() *AdminController {
	return &AdminController{}
}

// WelcomeAdmin
// @Summary      Welcome Admin
// @Description  Simple welcome message for admin
// @Tags         admin
// @Accept       plain
// @Produce      plain
// @Success      200  {string}  string "Welcome, Admin!"
// @Router       /api/admin [get]
func (ctrl *AdminController) WelcomeAdmin(c *fiber.Ctx) error {
	return c.SendString("Welcome, Admin!")
}
