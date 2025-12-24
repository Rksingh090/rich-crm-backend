package middleware

import (
	"context"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

// PermissionMiddleware checks if user has specific permission for a module/action
func PermissionMiddleware(roleService service.RoleService, moduleName string, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get roles from context (set by AuthMiddleware)
		rolesInterface := c.Locals("roles")
		if rolesInterface == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: No roles assigned",
			})
		}

		roles, ok := rolesInterface.([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: Invalid roles format",
			})
		}

		// Check if any of the user's roles has the required permission
		hasPermission, err := roleService.CheckModulePermission(context.Background(), roles, moduleName, permission)
		if err != nil || !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: Insufficient permissions for this action",
			})
		}

		return c.Next()
	}
}

// RequirePermission is a helper to create permission middleware
func RequirePermission(roleService service.RoleService, moduleName string, permission string) fiber.Handler {
	return PermissionMiddleware(roleService, moduleName, permission)
}
