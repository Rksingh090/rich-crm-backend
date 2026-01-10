package permission

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PermissionController struct {
	PermissionService PermissionService
}

func NewPermissionController(permissionService PermissionService) *PermissionController {
	return &PermissionController{
		PermissionService: permissionService,
	}
}

// CreatePermission godoc
// @Summary      Create a new permission
// @Description  Create a new permission assignment for a role
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        permission body Permission true "Permission object"
// @Success      201  {object} Permission
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to create permission"
// @Router       /api/permissions [post]
func (ctrl *PermissionController) CreatePermission(c *fiber.Ctx) error {
	var permission Permission
	if err := c.BodyParser(&permission); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	created, err := ctrl.PermissionService.CreatePermission(c.UserContext(), &permission)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// GetPermissionsByRole godoc
// @Summary      Get permissions for a role
// @Description  Get all permissions assigned to a specific role
// @Tags         permissions
// @Produce      json
// @Param        roleId path string true "Role ID"
// @Success      200  {array} Permission
// @Failure      500  {string} string "Failed to get permissions"
// @Router       /api/permissions/role/{roleId} [get]
func (ctrl *PermissionController) GetPermissionsByRole(c *fiber.Ctx) error {
	roleID := c.Params("roleId")

	permissions, err := ctrl.PermissionService.GetPermissionsByRole(c.UserContext(), roleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(permissions)
}

// GetPermissionsByResource godoc
// @Summary      Get permissions for a resource
// @Description  Get all permissions for a specific resource
// @Tags         permissions
// @Produce      json
// @Param        type query string true "Resource type"
// @Param        id query string true "Resource ID"
// @Success      200  {array} Permission
// @Failure      500  {string} string "Failed to get permissions"
// @Router       /api/permissions/resource [get]
func (ctrl *PermissionController) GetPermissionsByResource(c *fiber.Ctx) error {
	resourceType := c.Query("type")
	resourceID := c.Query("id")

	permissions, err := ctrl.PermissionService.GetPermissionsByResource(c.UserContext(), resourceType, resourceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(permissions)
}

// UpdatePermission godoc
// @Summary      Update a permission
// @Description  Update an existing permission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        id path string true "Permission ID"
// @Param        permission body Permission true "Permission object"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to update permission"
// @Router       /api/permissions/{id} [put]
func (ctrl *PermissionController) UpdatePermission(c *fiber.Ctx) error {
	id := c.Params("id")

	var permission Permission
	if err := c.BodyParser(&permission); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.PermissionService.UpdatePermission(c.UserContext(), id, &permission); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Permission updated successfully",
	})
}

// DeletePermission godoc
// @Summary      Delete a permission
// @Description  Delete a permission
// @Tags         permissions
// @Produce      json
// @Param        id path string true "Permission ID"
// @Success      200  {object} map[string]string
// @Failure      500  {string} string "Failed to delete permission"
// @Router       /api/permissions/{id} [delete]
func (ctrl *PermissionController) DeletePermission(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.PermissionService.DeletePermission(c.UserContext(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Permission deleted successfully",
	})
}

// AssignResourceToRole godoc
// @Summary      Assign a resource to a role
// @Description  Assign a resource to a role with specific actions and conditions
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body AssignResourceRequest true "Assignment request"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to assign resource"
// @Router       /api/permissions/assign [post]
func (ctrl *PermissionController) AssignResourceToRole(c *fiber.Ctx) error {
	var req AssignResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.PermissionService.AssignResourceToRole(c.UserContext(), req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Resource assigned to role successfully",
	})
}

// RevokeResourceFromRole godoc
// @Summary      Revoke a resource from a role
// @Description  Remove a resource assignment from a role
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request body RevokeResourceRequest true "Revoke request"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to revoke resource"
// @Router       /api/permissions/revoke [post]
func (ctrl *PermissionController) RevokeResourceFromRole(c *fiber.Ctx) error {
	var req RevokeResourceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.PermissionService.RevokeResourceFromRole(c.UserContext(), req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Resource revoked from role successfully",
	})
}

// GetUserEffectivePermissions godoc
// @Summary      Get effective permissions for a user
// @Description  Get aggregated permissions from all user's roles
// @Tags         permissions
// @Produce      json
// @Param        userId path string true "User ID"
// @Success      200  {object} map[string]Permission
// @Failure      400  {string} string "Invalid user ID"
// @Failure      500  {string} string "Failed to get permissions"
// @Router       /api/permissions/user/{userId}/effective [get]
func (ctrl *PermissionController) GetUserEffectivePermissions(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User context missing",
		})
	}
	userId, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID format",
		})
	}

	permissions, err := ctrl.PermissionService.GetUserEffectivePermissions(c.UserContext(), userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(permissions)
}
