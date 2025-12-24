package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type RoleController struct {
	Service service.RoleService
}

func NewRoleController(service service.RoleService) *RoleController {
	return &RoleController{Service: service}
}

// ListRoles godoc
// @Summary      List all roles
// @Description  Get all roles with their permissions
// @Tags         roles
// @Produce      json
// @Success      200  {array}   models.Role
// @Failure      500  {string}  string
// @Router       /roles [get]
func (c *RoleController) ListRoles(ctx *fiber.Ctx) error {
	roles, err := c.Service.ListRoles(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch roles",
		})
	}
	return ctx.JSON(roles)
}

// GetRole godoc
// @Summary      Get role by ID
// @Description  Get a specific role with its permissions
// @Tags         roles
// @Produce      json
// @Param        id   path      string  true  "Role ID"
// @Success      200  {object}  models.Role
// @Failure      404  {string}  string
// @Failure      500  {string}  string
// @Router       /roles/{id} [get]
func (c *RoleController) GetRole(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	role, err := c.Service.GetRoleByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}
	return ctx.JSON(role)
}

// CreateRole godoc
// @Summary      Create a new role
// @Description  Create a new role with module permissions
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        role  body      models.Role  true  "Role data"
// @Success      201   {object}  models.Role
// @Failure      400   {string}  string
// @Failure      500   {string}  string
// @Router       /roles [post]
func (c *RoleController) CreateRole(ctx *fiber.Ctx) error {
	var role models.Role
	if err := ctx.BodyParser(&role); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	createdRole, err := c.Service.CreateRole(ctx.Context(), &role)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create role",
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(createdRole)
}

// UpdateRole godoc
// @Summary      Update a role
// @Description  Update role permissions
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id    path      string       true  "Role ID"
// @Param        role  body      models.Role  true  "Role data"
// @Success      200   {object}  models.Role
// @Failure      400   {string}  string
// @Failure      404   {string}  string
// @Failure      500   {string}  string
// @Router       /roles/{id} [put]
func (c *RoleController) UpdateRole(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var role models.Role
	if err := ctx.BodyParser(&role); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := c.Service.UpdateRole(ctx.Context(), id, &role); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update role",
		})
	}

	updatedRole, _ := c.Service.GetRoleByID(ctx.Context(), id)
	return ctx.JSON(updatedRole)
}

// DeleteRole godoc
// @Summary      Delete a role
// @Description  Delete a role (cannot delete system roles)
// @Tags         roles
// @Produce      json
// @Param        id   path      string  true  "Role ID"
// @Success      200  {string}  string
// @Failure      400  {string}  string
// @Failure      500  {string}  string
// @Router       /roles/{id} [delete]
func (c *RoleController) DeleteRole(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteRole(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Role deleted successfully",
	})
}
