package routes

import (
	"net/http"

	"go-crm/internal/middleware"
	"go-crm/internal/service"
)

func RegisterAdminRoutes(mux *http.ServeMux, roleService service.RoleService, skipAuth bool) {
	// RBAC Protected Route
	adminOnly := middleware.RequirePermission(roleService, skipAuth, "admin:access", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome, Admin!"))
	})
	mux.Handle("/admin", middleware.AuthMiddleware(skipAuth)(adminOnly))
}
