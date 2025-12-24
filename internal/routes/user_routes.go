package routes

import (
	"go-crm/internal/handlers"
	"go-crm/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func RegisterUserRoutes(r chi.Router, userHandler *handlers.UserHandler, skipAuth bool) {
	r.Route("/users", func(r chi.Router) {
		// Apply auth middleware to all user routes
		r.Use(middleware.AuthMiddleware(skipAuth))

		// List users
		r.Get("/", userHandler.ListUsers)

		// Specific user operations
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", userHandler.GetUser)
			r.Put("/", userHandler.UpdateUser)
			r.Delete("/", userHandler.DeleteUser)

			// User sub-resources
			r.Put("/roles", userHandler.UpdateUserRoles)
			r.Put("/status", userHandler.UpdateUserStatus)
		})
	})
}
