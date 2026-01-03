package main

import (
	"context"
	"fmt"
	"go-crm/internal/api"
	"go-crm/internal/config"
	"go-crm/internal/controllers"
	"go-crm/internal/database"
	"go-crm/internal/logger"
	"go-crm/internal/middleware"
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

	// Use custom CORS middleware
	app.Use(middleware.CORSMiddleware())

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
			database.NewDatabase,

			// Initialize Repository
			repository.NewFileRepository,
			repository.NewAuditRepository,
			repository.NewModuleRepository,
			repository.NewUserRepository,
			repository.NewRecordRepository,
			repository.NewRoleRepository,
			repository.NewApprovalRepository,
			repository.NewReportRepository,
			repository.NewAutomationRepository,
			repository.NewSettingsRepository,
			repository.NewTicketRepository,
			repository.NewSLAPolicyRepository,
			repository.NewTicketCommentRepository,
			repository.NewEscalationRuleRepository,
			repository.NewGroupRepository,
			repository.NewNotificationRepository,
			repository.NewWebhookRepository,
			repository.NewExtensionRepository,
			repository.NewSyncSettingRepository,
			repository.NewSyncLogRepository,
			repository.NewChartRepository,
			repository.NewDashboardRepository,
			repository.NewImportRepository,
			repository.NewBulkOperationRepository,
			repository.NewSavedFilterRepository,
			repository.NewCronRepository,
			repository.NewEmailTemplateRepository,
			repository.NewDataSourceRepository,
			repository.NewMetricRepository,
			service.NewAuditService,
			service.NewAuthService,
			service.NewRoleService,
			service.NewModuleService,
			service.NewRecordService,
			service.NewUserService,
			service.NewFileService,
			service.NewApprovalService,
			service.NewSettingsService,
			service.NewEmailService,
			service.NewReportService,
			service.NewActionExecutor,
			service.NewAutomationService,
			service.NewTicketService,
			service.NewSLAService,
			service.NewEscalationService,
			service.NewGroupService,
			service.NewNotificationService,
			service.NewWebhookService,
			service.NewExtensionService,
			service.NewSyncService,
			service.NewSearchService,
			service.NewChartService,
			service.NewActivityService,
			service.NewDashboardService,
			service.NewImportService,
			service.NewBulkOperationService,
			service.NewSavedFilterService,
			service.NewCronService,
			service.NewEmailTemplateService,
			service.NewDataSourceService,
			service.NewAnalyticsService,

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
			controllers.NewApprovalController,
			controllers.NewReportController,
			controllers.NewAutomationController,
			controllers.NewSettingsController,
			controllers.NewTicketController,
			controllers.NewSLAMetricsController,
			controllers.NewGroupController,
			controllers.NewNotificationController,
			controllers.NewWebhookController,
			controllers.NewExtensionController,
			controllers.NewSyncController,
			controllers.NewSearchController,
			controllers.NewChartController,
			controllers.NewActivityController,
			controllers.NewDashboardController,
			controllers.NewImportController,
			controllers.NewBulkOperationController,
			controllers.NewSavedFilterController,
			controllers.NewCronController,
			controllers.NewEmailTemplateController,
			controllers.NewDataSourceController,
			controllers.NewAnalyticsController,

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
			AsRoute(api.NewApprovalApi),
			AsRoute(api.NewReportApi),
			AsRoute(api.NewAutomationApi),
			AsRoute(api.NewSettingsApi),
			AsRoute(api.NewTicketApi),
			AsRoute(api.NewGroupApi),
			AsRoute(api.NewNotificationApi),
			AsRoute(api.NewWebhookApi),
			AsRoute(api.NewExtensionApi),
			AsRoute(api.NewSyncApi),
			AsRoute(api.NewSearchApi),
			AsRoute(api.NewChartApi),
			AsRoute(api.NewActivityApi),
			AsRoute(api.NewDashboardApi),
			AsRoute(api.NewImportApi),
			AsRoute(api.NewBulkOperationApi),
			AsRoute(api.NewSavedFilterApi),
			AsRoute(api.NewCronApi),
			AsRoute(api.NewEmailTemplateApi),
			AsRoute(api.NewDataSourceApi),
			AsRoute(api.NewAnalyticsApi),
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(
			// Register Routes & Start
			RegisterSwagger,
			RegisterAllRoutesWithAnnotation,
			StartServer,
			func(lc fx.Lifecycle, cronService service.CronService) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						return cronService.InitializeScheduler(ctx)
					},
					OnStop: func(ctx context.Context) error {
						return cronService.StopScheduler()
					},
				})
			},
		),
	)

	app.Run()
}
