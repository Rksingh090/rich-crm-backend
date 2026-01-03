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
// GetEmailConfig godoc
// @Summary Get email configuration
// @Description Get the current email settings
// @Tags settings
// @Produce json
// @Success 200 {object} EmailConfig
// @Failure 500 {object} map[string]interface{}
// @Router /api/settings/email [get]
func (c *SettingsController) GetEmailConfig(ctx *fiber.Ctx) error {
	config, err := c.Service.GetEmailConfig(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if config == nil {
		return ctx.JSON(fiber.Map{})
	}
	return ctx.JSON(config)
}

// UpdateEmailConfig godoc
// UpdateEmailConfig godoc
// @Summary Update email configuration
// @Description Update the email settings
// @Tags settings
// @Accept json
// @Produce json
// @Param config body EmailConfig true "Email Configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/settings/email [put]
func (c *SettingsController) UpdateEmailConfig(ctx *fiber.Ctx) error {
	var config EmailConfig
	if err := ctx.BodyParser(&config); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := c.Service.UpdateEmailConfig(ctx.UserContext(), config); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Settings updated successfully"})
}

// GetGeneralConfig godoc
// GetGeneralConfig godoc
// @Summary Get general configuration
// @Description Get the general system settings
// @Tags settings
// @Produce json
// @Success 200 {object} GeneralConfig
// @Failure 500 {object} map[string]interface{}
// @Router /api/settings/general [get]
func (c *SettingsController) GetGeneralConfig(ctx *fiber.Ctx) error {
	config, err := c.Service.GetGeneralConfig(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(config)
}

// UpdateGeneralConfig godoc
// UpdateGeneralConfig godoc
// @Summary Update general configuration
// @Description Update the general system settings
// @Tags settings
// @Accept json
// @Produce json
// @Param config body GeneralConfig true "General Configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/settings/general [put]
func (ctrl *SettingsController) UpdateGeneralConfig(c *fiber.Ctx) error {
	var config GeneralConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateGeneralConfig(c.UserContext(), config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating general settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "General settings updated successfully",
	})
}

// GetFileSharingConfig godoc
// GetFileSharingConfig godoc
// @Summary Get file sharing configuration
// @Description Get the file sharing settings
// @Tags settings
// @Produce json
// @Success 200 {object} FileSharingConfig
// @Failure 500 {object} map[string]interface{}
// @Router /api/settings/file-sharing [get]
func (ctrl *SettingsController) GetFileSharingConfig(c *fiber.Ctx) error {
	config, err := ctrl.Service.GetFileSharingConfig(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving file sharing settings",
		})
	}

	return c.JSON(config)
}

// UpdateFileSharingConfig godoc
// UpdateFileSharingConfig godoc
// @Summary Update file sharing configuration
// @Description Update the file sharing settings
// @Tags settings
// @Accept json
// @Produce json
// @Param config body FileSharingConfig true "File Sharing Configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/settings/file-sharing [put]
func (ctrl *SettingsController) UpdateFileSharingConfig(c *fiber.Ctx) error {
	var config FileSharingConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.UpdateFileSharingConfig(c.UserContext(), config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating file sharing settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "File sharing settings updated successfully",
	})
}
