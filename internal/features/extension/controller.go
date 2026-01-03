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
func (ctrl *ExtensionController) ListExtensions(c *fiber.Ctx) error {
	installed := c.Query("installed") == "true"
	exts, err := ctrl.Service.ListExtensions(c.Context(), installed)
	if err != nil {
		return err
	}
	return c.JSON(exts)
}

// GetExtension godoc
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

// InstallExtension godoc
func (ctrl *ExtensionController) InstallExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.InstallExtension(c.Context(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "Extension installed successfully"})
}

// UninstallExtension godoc
func (ctrl *ExtensionController) UninstallExtension(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.Service.UninstallExtension(c.Context(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "Extension uninstalled successfully"})
}

// CreateExtension godoc
func (ctrl *ExtensionController) CreateExtension(c *fiber.Ctx) error {
	var ext Extension
	if err := c.BodyParser(&ext); err != nil {
		return err
	}
	if err := ctrl.Service.CreateExtension(c.Context(), &ext); err != nil {
		return err
	}
	return c.Status(201).JSON(ext)
}
