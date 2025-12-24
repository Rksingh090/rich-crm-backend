package middleware

import (
	"slices"

	"go-crm/internal/service"
	"go-crm/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// RequirePermission checks if the user has a specific permission
func RequirePermission(roleService service.RoleService, skipAuth bool, requiredPermission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if skipAuth {
			return c.Next()
		}

		claims, ok := c.Locals(utils.UserClaimsKey).(*utils.UserClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Check permissions via service
		permissions, err := roleService.GetPermissionsForRoles(c.Context(), claims.Roles)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error",
			})
		}

		if !slices.Contains(permissions, requiredPermission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: Insufficient permissions",
			})
		}

		return c.Next()
	}
}
