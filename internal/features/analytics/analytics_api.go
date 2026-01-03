package analytics

import (
	"github.com/gofiber/fiber/v2"
)

type AnalyticsApi struct {
	Controller *AnalyticsController
}

func NewAnalyticsApi(controller *AnalyticsController) *AnalyticsApi {
	return &AnalyticsApi{Controller: controller}
}

func (a *AnalyticsApi) Setup(app *fiber.App) {
	analytics := app.Group("/api/analytics")

	// Metrics
	metrics := analytics.Group("/metrics")
	metrics.Get("/", a.Controller.ListMetrics)
	metrics.Post("/", a.Controller.CreateMetric)
	metrics.Get("/:id", a.Controller.GetMetric)
	metrics.Put("/:id", a.Controller.UpdateMetric)
	metrics.Delete("/:id", a.Controller.DeleteMetric)
	metrics.Post("/:id/calculate", a.Controller.CalculateMetric)
	metrics.Get("/:id/history", a.Controller.GetMetricHistory)

	// Dashboard analytics
	dashboards := analytics.Group("/dashboards")
	dashboards.Get("/:id/metrics", a.Controller.GetDashboardMetrics)
}
