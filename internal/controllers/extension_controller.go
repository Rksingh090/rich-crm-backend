package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ExtensionController struct {
	Service service.ExtensionService
}

func NewExtensionController(service service.ExtensionService) *ExtensionController {
	return &ExtensionController{
		Service: service,
	}
}

// @Summary      List Extensions
// @Description  Get all available marketplace extensions
// @Tags         extensions
// @Accept       json
// @Produce      json
// @Param        installed  query     bool    false  "Only show installed extensions"
// @Success      200        {array}   models.Extension
// @Router       /api/extensions [get]
func (ctrl *ExtensionController) ListExtensions(c *fiber.Ctx) error {
	installed := c.Query("installed") == "true"
	exts, err := ctrl.Service.ListExtensions(c.Context(), installed)
	if err != nil {
		return err
	}
	return c.JSON(exts)
}

// @Summary      Get Extension
// @Description  Get a specific extension by ID
// @Tags         extensions
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Extension ID"
// @Success      200  {object}  models.Extension
// @Router       /api/extensions/{id} [get]
func (ctrl *ExtensionController) GetExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	ext, err := ctrl.Service.GetExtension(c.Context(), id)
	if err != nil {
		return err
	}
	if ext == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Extension not found"})
	}
	return c.JSON(ext)
}

// @Summary      Install Extension
// @Description  Install an extension by ID
// @Tags         extensions
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Extension ID"
// @Success      200  {object}  map[string]string
// @Router       /api/extensions/{id}/install [post]
func (ctrl *ExtensionController) InstallExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.InstallExtension(c.Context(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "Extension installed successfully"})
}

// @Summary      Uninstall Extension
// @Description  Uninstall an extension by ID
// @Tags         extensions
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Extension ID"
// @Success      200  {object}  map[string]string
// @Router       /api/extensions/{id}/uninstall [post]
func (ctrl *ExtensionController) UninstallExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.UninstallExtension(c.Context(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "Extension uninstalled successfully"})
}

// @Summary      Create Extension
// @Description  Create a new marketplace extension (Admin only)
// @Tags         extensions
// @Accept       json
// @Produce      json
// @Param        extension  body      models.Extension  true  "Extension Data"
// @Success      201        {object}  models.Extension
// @Router       /api/extensions [post]
func (ctrl *ExtensionController) CreateExtension(c *fiber.Ctx) error {
	var ext models.Extension
	if err := c.BodyParser(&ext); err != nil {
		return err
	}
	if err := ctrl.Service.CreateExtension(c.Context(), &ext); err != nil {
		return err
	}
	return c.Status(201).JSON(ext)
}
