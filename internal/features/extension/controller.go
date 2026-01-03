package extension

import (
	"github.com/gofiber/fiber/v2"
)

type ExtensionController struct {
	Service ExtensionService
}

func NewExtensionController(service ExtensionService) *ExtensionController {
	return &ExtensionController{
		Service: service,
	}
}

// ListExtensions godoc
// ListExtensions godoc
// @Summary List extensions
// @Description List extensions with optional installation filter
// @Tags extensions
// @Produce json
// @Param installed query boolean false "Filter by installed status"
// @Success 200 {array} Extension
// @Failure 500 {object} map[string]interface{}
// @Router /api/extensions [get]
func (ctrl *ExtensionController) ListExtensions(c *fiber.Ctx) error {
	installed := c.Query("installed") == "true"
	exts, err := ctrl.Service.ListExtensions(c.UserContext(), installed)
	if err != nil {
		return err
	}
	return c.JSON(exts)
}

// GetExtension godoc
// GetExtension godoc
// @Summary Get extension
// @Description Get an extension by ID
// @Tags extensions
// @Produce json
// @Param id path string true "Extension ID"
// @Success 200 {object} Extension
// @Failure 404 {object} map[string]interface{}
// @Router /api/extensions/{id} [get]
func (ctrl *ExtensionController) GetExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	ext, err := ctrl.Service.GetExtension(c.UserContext(), id)
	if err != nil {
		return err
	}
	if ext == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Extension not found"})
	}
	return c.JSON(ext)
}

// InstallExtension godoc
// InstallExtension godoc
// @Summary Install extension
// @Description Install an extension by ID
// @Tags extensions
// @Produce json
// @Param id path string true "Extension ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/extensions/{id}/install [post]
func (ctrl *ExtensionController) InstallExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.InstallExtension(c.UserContext(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "Extension installed successfully"})
}

// UninstallExtension godoc
// UninstallExtension godoc
// @Summary Uninstall extension
// @Description Uninstall an extension by ID
// @Tags extensions
// @Produce json
// @Param id path string true "Extension ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/extensions/{id}/uninstall [post]
func (ctrl *ExtensionController) UninstallExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.UninstallExtension(c.UserContext(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "Extension uninstalled successfully"})
}

// CreateExtension godoc
// CreateExtension godoc
// @Summary Create extension
// @Description Create a new extension
// @Tags extensions
// @Accept json
// @Produce json
// @Param extension body Extension true "Extension Details"
// @Success 201 {object} Extension
// @Failure 500 {object} map[string]interface{}
// @Router /api/extensions [post]
func (ctrl *ExtensionController) CreateExtension(c *fiber.Ctx) error {
	var ext Extension
	if err := c.BodyParser(&ext); err != nil {
		return err
	}
	if err := ctrl.Service.CreateExtension(c.UserContext(), &ext); err != nil {
		return err
	}
	return c.Status(201).JSON(ext)
}
