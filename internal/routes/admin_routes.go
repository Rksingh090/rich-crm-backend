package routes

import (
	"go-crm/internal/middleware"
	"go-crm/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterAdminRoutes(r chi.Router, roleService service.RoleService, skipAuth bool) {
	// RBAC Protected Route
	adminOnly := middleware.RequirePermission(roleService, skipAuth, "admin:access", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome, Admin!"))
	})
	r.With(middleware.AuthMiddleware(skipAuth)).Get("/admin", adminOnly)
}
