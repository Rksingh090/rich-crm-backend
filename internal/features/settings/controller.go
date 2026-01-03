package settings

import (
	"github.com/gofiber/fiber/v2"
)

type SettingsController struct {
	Service SettingsService
}

func NewSettingsController(service SettingsService) *SettingsController {
	return &SettingsController{
		Service: service,
	}
}

// GetEmailConfig godoc
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

// UpdateEmailConfig godoc
func (c *SettingsController) UpdateEmailConfig(ctx *fiber.Ctx) error {
	var config EmailConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.UpdateEmailConfig(ctx.Context(), config); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Settings updated successfully"})
}

// GetGeneralConfig godoc
func (c *SettingsController) GetGeneralConfig(ctx *fiber.Ctx) error {
	config, err := c.Service.GetGeneralConfig(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(config)
}

// UpdateGeneralConfig godoc
func (ctrl *SettingsController) UpdateGeneralConfig(c *fiber.Ctx) error {
	var config GeneralConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateGeneralConfig(c.Context(), config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating general settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "General settings updated successfully",
	})
}

// GetFileSharingConfig godoc
func (ctrl *SettingsController) GetFileSharingConfig(c *fiber.Ctx) error {
	config, err := ctrl.Service.GetFileSharingConfig(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving file sharing settings",
		})
	}

	return c.JSON(config)
}

// UpdateFileSharingConfig godoc
func (ctrl *SettingsController) UpdateFileSharingConfig(c *fiber.Ctx) error {
	var config FileSharingConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateFileSharingConfig(c.Context(), config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating file sharing settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "File sharing settings updated successfully",
	})
}
