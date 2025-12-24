package middleware

import (
	"go-crm/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates JWT tokens and injects user claims into context
func AuthMiddleware(skipAuth bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if skipAuth {
			// Inject dummy context for dev
			dummyClaims := &utils.UserClaims{
				UserID: "dev-admin-id",
				Roles:  []string{"admin"}, // Changed to admin for dev mode
			}
			c.Locals(utils.UserClaimsKey, dummyClaims)
			c.Locals("userID", dummyClaims.UserID)
			c.Locals("roles", dummyClaims.Roles)
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		// Extract token from "Bearer <token>"
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		token := authHeader[7:]
		claims, err := utils.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Store claims and also set userID and roles for other middleware
		c.Locals(utils.UserClaimsKey, claims)
		c.Locals("userID", claims.UserID)
		c.Locals("roles", claims.Roles)
		return c.Next()
	}
}
