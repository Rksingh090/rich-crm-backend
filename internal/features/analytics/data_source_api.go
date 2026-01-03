package analytics

import (
	"github.com/gofiber/fiber/v2"
)

type DataSourceApi struct {
	Controller *DataSourceController
}

func NewDataSourceApi(controller *DataSourceController) *DataSourceApi {
	return &DataSourceApi{Controller: controller}
}

func (a *DataSourceApi) Setup(app *fiber.App) {
	ds := app.Group("/api/data-sources")

	// Data source management
	ds.Get("/", a.Controller.ListDataSources)
	ds.Post("/", a.Controller.CreateDataSource)
	ds.Get("/:id", a.Controller.GetDataSource)
	ds.Put("/:id", a.Controller.UpdateDataSource)
	ds.Delete("/:id", a.Controller.DeleteDataSource)
	ds.Post("/:id/test", a.Controller.TestConnection)

	// Data querying
	ds.Post("/query", a.Controller.QueryDataSource)
	ds.Get("/:id/schema/:module", a.Controller.GetSchema)

	// Bulk operations
	ds.Post("/query/multi", a.Controller.QueryMultipleSources)
}
