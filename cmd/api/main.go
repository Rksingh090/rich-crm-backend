package main

import (
	"fmt"
	"log"
	"net/http"

	"go-crm/docs"
	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/handlers"
	"go-crm/internal/middleware"
	"go-crm/internal/repository"
	"go-crm/internal/routes"
	"go-crm/internal/service"
)

// @title           Go CRM API
// @version         1.0
// @description     This is a sample CRM server.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.LoadConfig()

	// Update Swagger info dynamically
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", cfg.Port)

	database.Connect(cfg)

	// Repositories
	userRepo := repository.NewMongoUserRepository(database.DB)
	roleRepo := repository.NewMongoRoleRepository(database.DB)
	moduleRepo := repository.NewMongoModuleRepository(database.DB)
	recordRepo := repository.NewMongoRecordRepository(database.DB)
	fileRepo := repository.NewMongoFileRepository(database.DB)
	auditRepo := repository.NewAuditRepository(database.DB)

	// Services
	// Services
	auditService := service.NewAuditServiceImpl(auditRepo)
	authService := service.NewAuthService(userRepo, roleRepo, auditService)
	roleService := service.NewRoleServiceImpl(roleRepo, auditService)
	moduleService := service.NewModuleServiceImpl(moduleRepo, auditService)
	recordService := service.NewRecordServiceImpl(moduleRepo, recordRepo, fileRepo, auditService)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	moduleHandler := handlers.NewModuleHandler(moduleService)
	recordHandler := handlers.NewRecordHandler(recordService)
	fileHandler := handlers.NewFileHandler("./uploads", fileRepo)
	auditHandler := handlers.NewAuditHandler(auditService)

	// Routes
	r := routes.SetupRoutes(cfg, authHandler, roleService, moduleHandler, recordHandler, fileHandler, auditHandler)

	// Wrap with CORS middleware
	handler := middleware.CORSMiddleware(r)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
