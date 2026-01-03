package module

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleController struct {
	Service ModuleService
}

func NewModuleController(service ModuleService) *ModuleController {
	return &ModuleController{
		Service: service,
	}
}

// CreateModule godoc
func (ctrl *ModuleController) CreateModule(c *fiber.Ctx) error {
	var m Module
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.CreateModule(c.UserContext(), &m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Module created successfully",
	})
}

// ListModules godoc
func (ctrl *ModuleController) ListModules(c *fiber.Ctx) error {
	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	product := c.Query("product")

	modules, err := ctrl.Service.ListModules(c.UserContext(), userID, product)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch modules",
		})
	}

	return c.JSON(modules)
}

// GetModule godoc
func (ctrl *ModuleController) GetModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	m, err := ctrl.Service.GetModuleByName(c.UserContext(), name, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Module not found",
		})
	}

	return c.JSON(m)
}

// UpdateModule godoc
func (ctrl *ModuleController) UpdateModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var m Module
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	m.Name = name // Ensure name matches path

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := ctrl.Service.UpdateModule(c.UserContext(), &m, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	// ... (DeleteModule)
	return c.JSON(fiber.Map{
		"message": "Module updated successfully",
	})
}

// DeleteModule godoc
func (ctrl *ModuleController) DeleteModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := ctrl.Service.DeleteModule(c.UserContext(), name, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Module deleted successfully",
	})
}
