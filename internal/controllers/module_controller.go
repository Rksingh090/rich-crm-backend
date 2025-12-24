package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ModuleController struct {
	Service service.ModuleService
}

func NewModuleController(service service.ModuleService) *ModuleController {
	return &ModuleController{
		Service: service,
	}
}

// CreateModule godoc
// @Summary      Create a new module definition
// @Description  Create a dynamic module with fields
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        input body models.Module true "Module Definition"
// @Success      201  {string} string "Module created"
// @Failure      400  {string} string "Invalid request"
// @Router       /modules [post]
func (ctrl *ModuleController) CreateModule(c *fiber.Ctx) error {
	var module models.Module
	if err := c.BodyParser(&module); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.Service.CreateModule(c.Context(), &module); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Module created successfully",
	})
}

// ListModules godoc
// @Summary      List all modules
// @Description  Get a list of all defined modules
// @Tags         modules
// @Produce      json
// @Success      200  {array} models.Module
// @Router       /modules [get]
func (ctrl *ModuleController) ListModules(c *fiber.Ctx) error {
	modules, err := ctrl.Service.ListModules(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch modules",
		})
	}

	return c.JSON(modules)
}

// GetModule godoc
// @Summary      Get a module by name
// @Description  Get specific module definition including fields
// @Tags         modules
// @Produce      json
// @Param        name path string true "Module Name"
// @Success      200  {object} models.Module
// @Failure      404  {string} string "Module not found"
// @Router       /modules/{name} [get]
func (ctrl *ModuleController) GetModule(c *fiber.Ctx) error {
	name := c.Params("name")

	module, err := ctrl.Service.GetModuleByName(c.Context(), name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Module not found",
		})
	}

	return c.JSON(module)
}

// UpdateModule godoc
// @Summary      Update a module definition
// @Description  Update schema for a module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        input body models.Module true "Module Definition"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /modules/{name} [put]
func (ctrl *ModuleController) UpdateModule(c *fiber.Ctx) error {
	name := c.Params("name")

	var module models.Module
	if err := c.BodyParser(&module); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	module.Name = name // Ensure name matches path

	if err := ctrl.Service.UpdateModule(c.Context(), &module); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Module updated successfully",
	})
}

// DeleteModule godoc
// @Summary      Delete a module
// @Description  Delete a module and its definition
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /modules/{name} [delete]
func (ctrl *ModuleController) DeleteModule(c *fiber.Ctx) error {
	name := c.Params("name")

	if err := ctrl.Service.DeleteModule(c.Context(), name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Module deleted successfully",
	})
}
