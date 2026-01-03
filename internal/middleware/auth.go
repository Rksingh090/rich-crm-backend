package middleware

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates JWT tokens and injects user claims into context
func AuthMiddleware(skipAuth bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if skipAuth {
			// Inject dummy context for dev
			// Use a valid ObjectID format for development
			dummyClaims := &utils.UserClaims{
				UserID:   "678e9a1b2c3d4e5f6a7b8c9d", // Valid ObjectID format
				TenantID: "678e9a1b2c3d4e5f6a7b8c9e", // Valid ObjectID format
				Roles:    []string{"admin"},
				Groups:   []string{"admins", "managers"}, // For ABAC testing
			}
			c.Locals(utils.UserClaimsKey, dummyClaims)
			c.Locals("user_id", dummyClaims.UserID)
			c.Locals("tenant_id", dummyClaims.TenantID)
			c.Locals("roles", dummyClaims.Roles)
			c.Locals("groups", dummyClaims.Groups)

			// Set organization context
			ctx := context.WithValue(c.UserContext(), models.TenantIDKey, dummyClaims.TenantID)
			c.SetUserContext(ctx)

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
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("roles", claims.Roles)
		c.Locals("groups", claims.Groups)

		ctx := context.WithValue(c.UserContext(), models.TenantIDKey, claims.TenantID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}
