package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SettingsController struct {
	Service service.SettingsService
}

func NewSettingsController(service service.SettingsService) *SettingsController {
	return &SettingsController{
		Service: service,
	}
}

// @Summary Get Email Configuration
// @Description Get current SMTP settings
// @Tags settings
// @Produce json
// @Success 200 {object} models.EmailConfig
// @Failure 500 {object} map[string]string
// @Router /settings/email [get]
func (c *SettingsController) GetEmailConfig(ctx *fiber.Ctx) error {
	config, err := c.Service.GetEmailConfig(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if config == nil {
		return ctx.JSON(fiber.Map{})
	}
	return ctx.JSON(config)
}

// @Summary Update Email Configuration
// @Description Update SMTP settings
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /settings/email [put]
func (c *SettingsController) UpdateEmailConfig(ctx *fiber.Ctx) error {
	var config models.EmailConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.UpdateEmailConfig(ctx.Context(), config); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Settings updated successfully"})
}
