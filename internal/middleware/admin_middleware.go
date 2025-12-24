package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AdminMiddleware checks if the user has admin role
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user from context (set by AuthMiddleware)
		userID := c.Locals("userID")
		if userID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Get roles from context
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

		// Check if user has admin role
		isAdmin := false
		for _, role := range roles {
			if strings.ToLower(role) == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: Admin role required",
			})
		}

		return c.Next()
	}
}
