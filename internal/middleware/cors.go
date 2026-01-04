package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSMiddleware returns Fiber's built-in CORS middleware with custom config
func CORSMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000, http://localhost:3001, http://localhost:3002, http://localhost:8000",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS, PATCH",
		AllowHeaders:     "Content-Type,Authorization,X-Requested-With,X-Rich-Product",
		AllowCredentials: true,
	})
}
