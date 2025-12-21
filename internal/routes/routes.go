package routes

import (
	"net/http"

	_ "go-crm/docs" // Import generated docs
	"go-crm/internal/config"
	"go-crm/internal/handlers"
	"go-crm/internal/middleware"
	"go-crm/internal/service"

	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRoutes(cfg *config.Config, authHandler *handlers.AuthHandler, roleService service.RoleService, moduleHandler *handlers.ModuleHandler, recordHandler *handlers.RecordHandler, fileHandler *handlers.FileHandler, auditHandler *handlers.AuditHandler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handlers.HealthCheck)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Static Files (Uploads)
	// StripPrefix removes "/uploads/" from path before looking in "uploads" dir
	fs := http.FileServer(http.Dir("./uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// File Upload Handler
	mux.Handle("/upload", middleware.AuthMiddleware(cfg.SkipAuth)(http.HandlerFunc(fileHandler.UploadFile)))

	// Register Modules
	RegisterAuthRoutes(mux, authHandler, cfg.SkipAuth)
	RegisterAdminRoutes(mux, roleService, cfg.SkipAuth)
	RegisterModuleRoutes(mux, moduleHandler, recordHandler, cfg.SkipAuth)

	// Audit Logs
	mux.Handle("/audit-logs", middleware.AuthMiddleware(cfg.SkipAuth)(http.HandlerFunc(auditHandler.ListLogs)))

	// WebSocket Route
	mux.HandleFunc("/ws", handlers.HandleWebSocket)

	return mux
}
