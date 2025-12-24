package controllers

import (
	"github.com/gofiber/fiber/v2"
)

// 1. Define the Struct directly (Exported)
type AdminController struct {
}

// 2. Constructor returns the pointer to the struct (*HealthController)
func NewAdminController() *AdminController {
	return &AdminController{}
}

// Check godoc
// @Summary      Health Check
// @Description  Get system health status
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func (ctrl *AdminController) WelcomeAdmin(c *fiber.Ctx) error {
	return c.SendString("Welcome, Admin!")
}
