package routes

import (
	_ "go-crm/docs" // Import generated docs
	"go-crm/internal/config"
	"go-crm/internal/handlers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRoutes(cfg *config.Config, authHandler *handlers.AuthHandler, roleService service.RoleService, moduleHandler *handlers.ModuleHandler, recordHandler *handlers.RecordHandler, fileHandler *handlers.FileHandler, auditHandler *handlers.AuditHandler, userHandler *handlers.UserHandler) chi.Router {
	r := chi.NewRouter()

	// Health check
	r.Get("/health", handlers.HealthCheck)

	// Swagger UI
	r.Handle("/swagger/*", httpSwagger.WrapHandler)

	// Static Files (Uploads)
	fs := http.FileServer(http.Dir("./uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", fs))

	// File Upload Handler (Protected)
	r.With(middleware.AuthMiddleware(cfg.SkipAuth)).Post("/upload", fileHandler.UploadFile)

	// Register route modules
	RegisterAuthRoutes(r, authHandler, cfg.SkipAuth)
	RegisterAdminRoutes(r, roleService, cfg.SkipAuth)
	RegisterModuleRoutes(r, moduleHandler, recordHandler, cfg.SkipAuth)
	RegisterUserRoutes(r, userHandler, cfg.SkipAuth)

	// Audit Logs (Protected)
	r.With(middleware.AuthMiddleware(cfg.SkipAuth)).Get("/audit-logs", auditHandler.ListLogs)

	// WebSocket Route
	r.HandleFunc("/ws", handlers.HandleWebSocket)

	return r
}
