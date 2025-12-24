package routes

import (
	"go-crm/internal/handlers"
	"go-crm/internal/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterAuthRoutes(r chi.Router, authHandler *handlers.AuthHandler, skipAuth bool) {
	// Public Routes
	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	// Protected Route Example
	r.With(middleware.AuthMiddleware(skipAuth)).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("You are authenticated!"))
	})
}
