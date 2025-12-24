package api

import "github.com/gofiber/fiber/v2"

// Route is an interface for any module that wants to register endpoints
type Route interface {
	Setup(app *fiber.App)
}

// fiberRouteImpl is a helper struct to hold the app
// We will embed this in our specific route files
type FiberRoute struct {
	App *fiber.App
}
