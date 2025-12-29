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

// @Summary Get General Configuration
// @Description Get system general settings
// @Tags settings
// @Produce json
// @Success 200 {object} models.GeneralConfig
// @Failure 500 {object} map[string]string
// @Router /settings/general [get]
func (c *SettingsController) GetGeneralConfig(ctx *fiber.Ctx) error {
	config, err := c.Service.GetGeneralConfig(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(config)
}

// @Summary Update General Configuration
// @Description Update system general settings
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /settings/general [put]
func (ctrl *SettingsController) UpdateGeneralConfig(c *fiber.Ctx) error {
	var config models.GeneralConfig
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
// @Summary      Get file sharing configuration
// @Description  Get current file sharing settings
// @Tags         settings
// @Produce      json
// @Success      200 {object} models.FileSharingConfig
// @Router       /settings/file-sharing [get]
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
// @Summary      Update file sharing configuration
// @Description  Update file sharing settings (admin only)
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        config body models.FileSharingConfig true "File Sharing Configuration"
// @Success      200 {object} map[string]string
// @Router       /settings/file-sharing [put]
func (ctrl *SettingsController) UpdateFileSharingConfig(c *fiber.Ctx) error {
	var config models.FileSharingConfig
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
