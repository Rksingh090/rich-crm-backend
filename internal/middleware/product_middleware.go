package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

type ProductContextKey string

const ProductKey ProductContextKey = "product"

// ProductMiddleware extracts the X-Rich-Product header and adds it to the context
func ProductMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		product := c.Get("X-Rich-Product")
		if product != "" {
			// Add product to context
			ctx := context.WithValue(c.UserContext(), ProductKey, product)
			c.SetUserContext(ctx)
		}
		return c.Next()
	}
}
