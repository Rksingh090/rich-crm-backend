package middleware

import (
	"context"
	"go-crm/internal/common/models"

	"github.com/gofiber/fiber/v2"
)

// AppMiddleware extracts the X-Rich-App header and adds it to the context
func AppMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		app := c.Get("X-Rich-App")
		if app != "" {
			// Add app to context
			ctx := context.WithValue(c.UserContext(), models.AppIDKey, app)
			c.SetUserContext(ctx)
		}
		return c.Next()
	}
}
