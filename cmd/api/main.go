package main

import (
	"context"
	"fmt"
	"go-crm/internal/api"
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/database"
	"go-crm/internal/logger"
	"go-crm/internal/repository"
	"go-crm/internal/service"
	"log"

	_ "go-crm/docs" // Import swagger docs

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// NewFiberServer creates a new Fiber app instance
func NewFiberServer() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// CORS middleware - allow frontend at localhost:3000
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Set("Access-Control-Allow-Credentials", "true")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	})

	return app
}

// AsRoute is a helper function to reduce boilerplate.
// It tags the constructor so Fx knows to add it to the "routes" group.
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(api.Route)),           // Cast to Interface
		fx.ResultTags(`group:"routes"`), // Add to Group
	)
}

// RegisterAllRoutes takes the group "routes" (slice of interfaces)
// and calls Setup() on each one.
func RegisterAllRoutes(app *fiber.App, routes []api.Route) {
	log.Printf("Registering %d routes...\n", len(routes))
	for i, route := range routes {
		log.Printf("Setting up route %d: %T\n", i+1, route)
		route.Setup(app)
	}
	log.Println("All routes registered successfully")
}

// RegisterAllRoutesWithAnnotation wraps RegisterAllRoutes with fx annotations
var RegisterAllRoutesWithAnnotation = fx.Annotate(
	RegisterAllRoutes,
	fx.ParamTags(``, `group:"routes"`),
)

func RegisterSwagger(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}

// StartServer creates a lifecycle hook to start Fiber in a goroutine
// and shut it down when the app exits.
// StartServer now needs Config to know which port to listen on
func StartServer(lc fx.Lifecycle, app *fiber.App, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				port := fmt.Sprintf(":%s", cfg.Port)
				if err := app.Listen(port); err != nil {
					log.Fatalf("Server failed to start: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.Shutdown()
		},
	})
}

// @title           Microservice Demo API
// @version         1.0
// @description     This is a sample server using Fiber, Uber Fx, and GORM.
// @termsOfService  http://swagger.io/terms/

// @contact.name    API Support
// @contact.email   support@swagger.io

// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @host            localhost:8000
// @BasePath        /
func main() {
	app := fx.New(
		fx.Provide(
			// Load Config
			config.LoadConfig,

			// Initialize Logger
			logger.NewLogger,

			// Initialize Fiber Server
			NewFiberServer,

			// Initialize Database
			database.NewDatabase, // Returns *MssqlDB

			// Initialize Repository
			repository.NewFileRepository,
			repository.NewAuditRepository,
			repository.NewModuleRepository,
			repository.NewUserRepository,
			repository.NewRecordRepository,
			repository.NewRoleRepository,

			service.NewAuditService,
			service.NewAuthService,
			service.NewRoleService,
			service.NewModuleService,
			service.NewRecordService,
			service.NewUserService,

			// Initialize Controller
			controllers.NewAdminController,
			controllers.NewAuthController,
			controllers.NewRoleController,
			controllers.NewModuleController,
			controllers.NewRecordController,
			controllers.NewUserController,
			controllers.NewFileController,
			controllers.NewAuditController,
			controllers.NewDebugController,

			// Initialize API Routes
			AsRoute(api.NewAdminApi),
			AsRoute(api.NewAuthApi),
			AsRoute(api.NewRoleApi),
			AsRoute(api.NewModuleApi),
			AsRoute(api.NewRecordApi),
			AsRoute(api.NewUserApi),
			AsRoute(api.NewFileApi),
			AsRoute(api.NewAuditApi),
			AsRoute(api.NewDebugApi),
			AsRoute(api.NewHealthApi),
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(
			// Register Routes & Start
			RegisterSwagger,
			RegisterAllRoutesWithAnnotation,
			StartServer,
		),
	)

	app.Run()
}
