package routes

import (
	"net/http"

	"go-crm/internal/handlers"
	"go-crm/internal/middleware"
)

func RegisterAuthRoutes(mux *http.ServeMux, authHandler *handlers.AuthHandler, skipAuth bool) {
	// Public Routes
	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)

	// Protected Routes
	// We wrap the handler with AuthMiddleware
	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("You are authenticated!"))
	})
	mux.Handle("/protected", middleware.AuthMiddleware(skipAuth)(protected))
}
