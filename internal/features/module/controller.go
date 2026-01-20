package module

import (
	"context"
	"go-crm/internal/common/models"

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
// @Description Creates a new module/entity for the specified application
// @Tags Modules
// @Accept json
// @Produce json
// @Param X-Rich-App header string true "Application identifier (e.g., crm, erp)"
// @Param module body models.Entity true "Module data"
// @Success 201 {object} map[string]string "Module created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or missing product header"
// @Router /modules [post]
// @Security BearerAuth
func (ctrl *ModuleController) CreateModule(c *fiber.Ctx) error {
	var m models.Entity
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	// Set Product from Header
	product := c.Get("X-Rich-App")
	if product != "" {
		m.App = models.App(product)
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product not found in header",
		})
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
// @Description Retrieves all modules for the current user and application
// @Tags Modules
// @Accept json
// @Produce json
// @Param X-Rich-App header string false "Application identifier (defaults to 'crm')"
// @Success 200 {array} models.Entity "List of modules"
// @Failure 500 {object} map[string]string "Failed to fetch modules"
// @Router /modules [get]
// @Security BearerAuth
func (ctrl *ModuleController) ListModules(c *fiber.Ctx) error {
	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	// Get product from header and add to context
	ctx := c.UserContext()
	product := c.Get("X-Rich-App", "crm")
	if product != "" {
		ctx = context.WithValue(ctx, models.AppIDKey, product)
	}

	modules, err := ctrl.Service.ListModules(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch modules",
		})
	}

	return c.JSON(modules)
}

// GetModule godoc
// @Summary Get a module by name
// @Description Retrieves a specific module by its name
// @Tags Modules
// @Accept json
// @Produce json
// @Param name path string true "Module name"
// @Success 200 {object} models.Entity "Module details"
// @Failure 404 {object} map[string]string "Module not found"
// @Router /modules/{name} [get]
// @Security BearerAuth
func (ctrl *ModuleController) GetModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	var m *models.Entity
	var err error

	if primitive.IsValidObjectID(name) {
		m, err = ctrl.Service.GetModuleByID(c.UserContext(), name, userID)
		// If found by ID, return it. If not found, fall through to Name check?
		// Actually typical use case: if it looks like ID, treat as ID. If fails, return error.
		// But "valid object id" might also be a valid name (unlikely but possible).
		// Let's assume if it looks like ID, we try ID. If it fails (not found), we COULD try Name but that might mask errors.
		// Given the Frontend explicitly sends ID, if we find it, great.
		if err == nil {
			return c.JSON(m)
		}
		// If error is NOT "not found", return error
		// If "not found", maybe fallthrough to name?
		// For now, let's keep it simple: Try ID. If invalid/not found, ALSO Try Name.
	}

	// Fallback or Primary: GetByName
	m, err = ctrl.Service.GetModuleByName(c.UserContext(), name, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(m)
}

// UpdateModule godoc
// @Summary Update a module
// @Description Updates an existing module by name
// @Tags Modules
// @Accept json
// @Produce json
// @Param name path string true "Module name"
// @Param module body models.Entity true "Updated module data"
// @Success 200 {object} map[string]string "Module updated successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Failed to update module"
// @Router /modules/{name} [put]
// @Security BearerAuth
func (ctrl *ModuleController) UpdateModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var m models.Entity
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
	return c.JSON(fiber.Map{
		"message": "Module updated successfully",
	})
}

// DeleteModule godoc
// @Summary Delete a module
// @Description Deletes a module by name
// @Tags Modules
// @Accept json
// @Produce json
// @Param name path string true "Module name"
// @Success 200 {object} map[string]string "Module deleted successfully"
// @Failure 500 {object} map[string]string "Failed to delete module"
// @Router /modules/{name} [delete]
// @Security BearerAuth
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
