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
// @Summary Create a new module
// @Description Create a new module definition
// @Tags modules
// @Accept json
// @Produce json
// @Param module body Module true "Module Definition"
// @Success 201 {object} map[string]string "Module created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or validation error"
// @Router /api/modules [post]
func (ctrl *ModuleController) CreateModule(c *fiber.Ctx) error {
	var m Module
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := ctrl.Service.CreateModule(c.UserContext(), &m, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Module created successfully",
	})
}

// ListModules godoc
// @Summary List all modules
// @Description List all available modules, optionally filtered by product
// @Tags modules
// @Accept json
// @Produce json
// @Param product query string false "Filter by product"
// @Success 200 {array} Module "List of modules"
// @Failure 500 {object} map[string]string "Failed to fetch modules"
// @Router /api/modules [get]
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
// @Summary Get a module by name
// @Description Get a module definition by its name
// @Tags modules
// @Accept json
// @Produce json
// @Param name path string true "Module Name"
// @Success 200 {object} Module "Module details"
// @Failure 404 {object} map[string]string "Module not found"
// @Router /api/modules/{name} [get]
func (ctrl *ModuleController) GetModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	m, err := ctrl.Service.GetModuleByName(c.UserContext(), name, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(m)
}

// UpdateModule godoc
// @Summary Update a module
// @Description Update an existing module definition
// @Tags modules
// @Accept json
// @Produce json
// @Param name path string true "Module Name"
// @Param module body Module true "Module Definition"
// @Success 200 {object} map[string]string "Module updated successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/modules/{name} [put]
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
// @Summary Delete a module
// @Description Delete a module definition
// @Tags modules
// @Accept json
// @Produce json
// @Param name path string true "Module Name"
// @Success 200 {object} map[string]string "Module deleted successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/modules/{name} [delete]
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
