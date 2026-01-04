package main

import (
	"context"
	"fmt"
	common_api "go-crm/internal/common/api"
	"go-crm/internal/config"
	"go-crm/internal/database"
	"go-crm/internal/features/activity"
	"go-crm/internal/features/admin"
	"go-crm/internal/features/analytics"
	"go-crm/internal/features/approval"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/auth"
	"go-crm/internal/features/automation"
	"go-crm/internal/features/bulk_operation"
	"go-crm/internal/features/chart"
	cron_feature "go-crm/internal/features/cron"
	"go-crm/internal/features/dashboard"
	"go-crm/internal/features/email"
	"go-crm/internal/features/email_template"
	"go-crm/internal/features/extension"
	"go-crm/internal/features/file"
	"go-crm/internal/features/group"
	import_feature "go-crm/internal/features/import"
	"go-crm/internal/features/module"
	"go-crm/internal/features/notification"
	"go-crm/internal/features/organization"
	"go-crm/internal/features/permission"
	"go-crm/internal/features/record"
	"go-crm/internal/features/report"
	"go-crm/internal/features/resource"
	"go-crm/internal/features/role"
	"go-crm/internal/features/saved_filter"
	"go-crm/internal/features/search"
	"go-crm/internal/features/settings"
	"go-crm/internal/features/sync"
	"go-crm/internal/features/system"
	"go-crm/internal/features/ticket"
	"go-crm/internal/features/user"
	"go-crm/internal/features/webhook"
	"go-crm/internal/logger"
	"go-crm/internal/middleware"
	"log"
	"time"

	_ "go-crm/docs" // Import swagger docs

	"github.com/gofiber/fiber/v2"
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

	// Add Product middleware to extract X-Rich-Product header
	app.Use(middleware.ProductMiddleware())

	return app
}

// AsRoute is a helper function to reduce boilerplate.
// It tags the constructor so Fx knows to add it to the "routes" group.
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(common_api.Route)),    // Cast to Interface
		fx.ResultTags(`group:"routes"`), // Add to Group
	)
}

// RegisterAllRoutes takes the group "routes" (slice of interfaces)
// and calls Setup() on each one.
func RegisterAllRoutes(app *fiber.App, routes []common_api.Route) {
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

// InitializeIndexes ensures that necessary database indexes are created
func InitializeIndexes(lc fx.Lifecycle, moduleRepo module.ModuleRepository, resourceRepo resource.ResourceRepository) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				// Use a background context with timeout for index creation
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := moduleRepo.EnsureIndexes(ctx); err != nil {
					log.Printf("Failed to ensure module indexes: %v", err)
				}
				if err := resourceRepo.EnsureIndexes(ctx); err != nil {
					log.Printf("Failed to ensure resource indexes: %v", err)
				}
			}()
			return nil
		},
	})
}

// resourceServiceAdapter adapts ResourceService to the interface expected by ModuleService
type resourceServiceAdapter struct {
	svc resource.ResourceService
}

func (a *resourceServiceAdapter) CreateResource(ctx context.Context, res interface{}) error {
	// Convert map to Resource struct
	resMap, ok := res.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid resource type")
	}

	r := &resource.Resource{}
	if v, ok := resMap["resource"].(string); ok {
		r.ResourceID = v
	}
	if v, ok := resMap["product"].(string); ok {
		r.Product = v
	}
	if v, ok := resMap["type"].(string); ok {
		r.Type = v
	}
	if v, ok := resMap["key"].(string); ok {
		r.Key = v
	}
	if v, ok := resMap["label"].(string); ok {
		r.Label = v
	}
	if v, ok := resMap["icon"].(string); ok {
		r.Icon = v
	}
	if v, ok := resMap["route"].(string); ok {
		r.Route = v
	}
	if v, ok := resMap["actions"].([]string); ok {
		r.Actions = v
	}
	if v, ok := resMap["configurable"].(bool); ok {
		r.Configurable = v
	}
	if v, ok := resMap["is_system"].(bool); ok {
		r.IsSystem = v
	}
	if v, ok := resMap["scope"].(string); ok {
		r.Scope = v
	}
	if v, ok := resMap["is_override"].(bool); ok {
		r.IsOverride = v
	}
	if v, ok := resMap["base_resource_id"].(string); ok {
		r.BaseResourceID = v
	}
	if ui, ok := resMap["ui"].(map[string]interface{}); ok {
		if v, ok := ui["sidebar"].(bool); ok {
			r.UI.Sidebar = v
		}
		if v, ok := ui["order"].(int); ok {
			r.UI.Order = v
		}
		if v, ok := ui["group"].(string); ok {
			r.UI.Group = v
		}
		if v, ok := ui["group_order"].(int); ok {
			r.UI.GroupOrder = v
		}
		if v, ok := ui["location"].(string); ok {
			r.UI.Location = v
		}
	}

	return a.svc.CreateResource(ctx, r)
}

func (a *resourceServiceAdapter) DeleteResource(ctx context.Context, resourceID string, userID string) error {
	return a.svc.DeleteResource(ctx, resourceID, userID)
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
			file.NewFileRepository,
			audit.NewAuditRepository,
			module.NewModuleRepository,
			organization.NewOrganizationRepository,
			user.NewUserRepository,
			record.NewRecordRepository,
			role.NewRoleRepository,
			approval.NewApprovalRepository,
			report.NewReportRepository,
			automation.NewAutomationRepository,
			settings.NewSettingsRepository,
			ticket.NewTicketRepository,
			ticket.NewSLAPolicyRepository,
			ticket.NewTicketCommentRepository,
			ticket.NewEscalationRuleRepository,
			group.NewGroupRepository,
			notification.NewNotificationRepository,
			webhook.NewWebhookRepository,
			webhook.NewWebhookLogRepository,
			extension.NewExtensionRepository,
			sync.NewSyncSettingRepository,
			sync.NewSyncLogRepository,
			chart.NewChartRepository,
			dashboard.NewDashboardRepository,
			email.NewEmailRepository,
			email_template.NewEmailTemplateRepository,
			bulk_operation.NewBulkOperationRepository,
			saved_filter.NewSavedFilterRepository,
			cron_feature.NewCronRepository,
			import_feature.NewImportRepository,
			analytics.NewMetricRepository,
			analytics.NewDataSourceRepository,
			resource.NewResourceRepository,
			permission.NewPermissionRepository,

			audit.NewAuditService,
			auth.NewAuthService,
			role.NewRoleService,
			module.NewModuleService,
			record.NewRecordService,
			user.NewUserService,
			file.NewFileService,
			group.NewGroupService,
			approval.NewApprovalService,
			settings.NewSettingsService,
			report.NewReportService,
			automation.NewActionExecutor,
			automation.NewAutomationService,
			ticket.NewTicketService,
			ticket.NewSLAService,
			ticket.NewEscalationService,
			notification.NewNotificationService,
			webhook.NewWebhookService,
			extension.NewExtensionService,
			sync.NewSyncService,
			search.NewSearchService,
			activity.NewActivityService,
			chart.NewChartService,
			dashboard.NewDashboardService,
			email.NewEmailService,
			email_template.NewEmailTemplateService,
			cron_feature.NewCronService,
			bulk_operation.NewBulkOperationService,
			import_feature.NewImportService,
			saved_filter.NewSavedFilterService,
			analytics.NewAnalyticsService,
			analytics.NewDataSourceService,
			resource.NewResourceService,
			permission.NewPermissionService,

			// Interface Adapters to break circular dependencies and satisfy Fx
			func(s approval.ApprovalService) record.ApprovalTrigger { return s },
			func(s automation.AutomationService) record.AutomationTrigger { return s },
			func(s role.RoleService) middleware.RoleService { return s },
			func(r user.UserRepository) audit.UserFinder { return r },
			func(s resource.ResourceService) interface {
				CreateResource(ctx context.Context, resource interface{}) error
				DeleteResource(ctx context.Context, resourceID string, userID string) error
			} {
				return &resourceServiceAdapter{svc: s}
			},

			// Initialize Controller
			admin.NewAdminController,
			auth.NewAuthController,
			role.NewRoleController,
			module.NewModuleController,
			record.NewRecordController,
			user.NewUserController,
			file.NewFileController,
			audit.NewAuditController,
			system.NewDebugController,
			system.NewWebSocketController,
			approval.NewApprovalController,
			report.NewReportController,
			automation.NewAutomationController,
			settings.NewSettingsController,
			ticket.NewTicketController,
			ticket.NewSLAMetricsController,
			group.NewGroupController,
			notification.NewNotificationController,
			webhook.NewWebhookController,
			extension.NewExtensionController,
			sync.NewSyncController,
			search.NewSearchController,
			activity.NewActivityController,
			chart.NewChartController,
			dashboard.NewDashboardController,
			email_template.NewEmailTemplateController,
			import_feature.NewImportController,
			bulk_operation.NewBulkOperationController,
			saved_filter.NewSavedFilterController,
			cron_feature.NewCronController,
			analytics.NewAnalyticsController,
			analytics.NewDataSourceController,
			resource.NewResourceController,
			permission.NewPermissionController,

			// Initialize API Routes
			AsRoute(admin.NewAdminApi),
			AsRoute(auth.NewAuthApi),
			AsRoute(role.NewRoleApi),
			AsRoute(module.NewModuleApi),
			AsRoute(record.NewRecordApi),
			AsRoute(user.NewUserApi),
			AsRoute(file.NewFileApi),
			AsRoute(audit.NewAuditApi),
			AsRoute(system.NewDebugApi),
			AsRoute(system.NewHealthApi),
			AsRoute(approval.NewApprovalApi),
			AsRoute(report.NewReportApi),
			AsRoute(automation.NewAutomationApi),
			AsRoute(settings.NewSettingsApi),
			AsRoute(ticket.NewTicketApi),
			AsRoute(group.NewGroupApi),
			AsRoute(notification.NewNotificationApi),
			AsRoute(webhook.NewWebhookApi),
			AsRoute(extension.NewExtensionApi),
			AsRoute(sync.NewSyncApi),
			AsRoute(search.NewSearchApi),
			AsRoute(chart.NewChartApi),
			AsRoute(activity.NewActivityApi),
			AsRoute(dashboard.NewDashboardApi),
			AsRoute(import_feature.NewImportApi),
			AsRoute(bulk_operation.NewBulkOperationApi),
			AsRoute(saved_filter.NewSavedFilterApi),
			AsRoute(cron_feature.NewCronApi),
			AsRoute(email_template.NewEmailTemplateApi),
			AsRoute(system.NewSwaggerApi),
			AsRoute(analytics.NewAnalyticsApi),
			AsRoute(analytics.NewDataSourceApi),
			AsRoute(resource.NewResourceApi),
			AsRoute(permission.NewPermissionApi),
			AsRoute(system.NewWebSocketApi),
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(
			// Register Routes & Start
			RegisterAllRoutesWithAnnotation,
			StartServer,
			func(lc fx.Lifecycle, cronService cron_feature.CronService) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						return cronService.InitializeScheduler(ctx)
					},
					OnStop: func(ctx context.Context) error {
						return cronService.StopScheduler()
					},
				})
			},
			InitializeIndexes,
		),
	)

	app.Run()
}
