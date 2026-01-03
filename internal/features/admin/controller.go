package admin

import (
	"fmt"

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

// HandleWebhook
// @Summary      Handle Webhook
// @Description  Receive and print webhook payload
// @Tags         admin
// @Accept       json
// @Produce      plain
// @Param        payload  body      string  true  "Webhook Payload"
// @Success      200      {string}  string  "Webhook handled successfully!"
// @Router       /api/admin/handle-webhook [post]
func (ctrl *AdminController) HandleWebhook(c *fiber.Ctx) error {
	body := c.Body()
	fmt.Println(string(body))

	return c.SendString("Webhook handled successfully!")
}
