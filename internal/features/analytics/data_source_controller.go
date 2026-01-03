package analytics

import (
	"go-crm/internal/connectors"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DataSourceController struct {
	Service DataSourceService
}

func NewDataSourceController(service DataSourceService) *DataSourceController {
	return &DataSourceController{Service: service}
}

// CreateDataSource creates a new data source
// @Summary Create data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param dataSource body DataSource true "Data Source"
// @Success 201 {object} DataSource
// @Router /api/data-sources [post]
func (c *DataSourceController) CreateDataSource(ctx *fiber.Ctx) error {
	var dataSource DataSource
	if err := ctx.BodyParser(&dataSource); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get user ID from context
	if userID, ok := ctx.Locals("user_id").(primitive.ObjectID); ok {
		dataSource.CreatedBy = userID
	}

	if err := c.Service.CreateDataSource(ctx.UserContext(), &dataSource); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(dataSource)
}

// GetDataSource retrieves a data source by ID
// @Summary Get data source
// @Tags data-sources
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} DataSource
// @Router /api/data-sources/{id} [get]
func (c *DataSourceController) GetDataSource(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	dataSource, err := c.Service.GetDataSource(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data source not found"})
	}

	return ctx.JSON(dataSource)
}

// ListDataSources lists all data sources
// @Summary List data sources
// @Tags data-sources
// @Produce json
// @Success 200 {array} DataSource
// @Router /api/data-sources [get]
func (c *DataSourceController) ListDataSources(ctx *fiber.Ctx) error {
	dataSources, err := c.Service.ListDataSources(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(dataSources)
}

// UpdateDataSource updates a data source
// @Summary Update data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param updates body map[string]interface{} true "Updates"
// @Success 200 {object} map[string]interface{}
// @Router /api/data-sources/{id} [put]
func (c *DataSourceController) UpdateDataSource(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var updates map[string]interface{}
	if err := ctx.BodyParser(&updates); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := c.Service.UpdateDataSource(ctx.UserContext(), id, updates); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Data source updated successfully"})
}

// DeleteDataSource deletes a data source
// @Summary Delete data source
// @Tags data-sources
// @Param id path string true "Data Source ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/data-sources/{id} [delete]
func (c *DataSourceController) DeleteDataSource(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.DeleteDataSource(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Data source deleted successfully"})
}

// TestConnection tests a data source connection
// @Summary Test data source connection
// @Tags data-sources
// @Param id path string true "Data Source ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/data-sources/{id}/test [post]
func (c *DataSourceController) TestConnection(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.Service.TestDataSource(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "status": "failed"})
	}

	return ctx.JSON(fiber.Map{"status": "success", "message": "Connection successful"})
}

// QueryDataSource executes a query on a data source
// @Summary Query data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param query body connectors.QueryRequest true "Query Request"
// @Success 200 {object} connectors.QueryResponse
// @Router /api/data-sources/query [post]
func (c *DataSourceController) QueryDataSource(ctx *fiber.Ctx) error {
	var query connectors.QueryRequest
	if err := ctx.BodyParser(&query); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	response, err := c.Service.QueryDataSource(ctx.UserContext(), query)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(response)
}

// GetSchema retrieves schema for a module/table
// @Summary Get data source schema
// @Tags data-sources
// @Param id path string true "Data Source ID"
// @Param module path string true "Module/Table Name"
// @Success 200 {object} connectors.SchemaInfo
// @Router /api/data-sources/{id}/schema/{module} [get]
func (c *DataSourceController) GetSchema(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	module := ctx.Params("module")

	schema, err := c.Service.GetDataSourceSchema(ctx.UserContext(), id, module)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(schema)
}

// QueryMultipleSources executes queries on multiple data sources
// @Summary Query multiple data sources
// @Tags data-sources
// @Accept json
// @Produce json
// @Param queries body []connectors.QueryRequest true "Query Requests"
// @Success 200 {object} map[string]connectors.QueryResponse
// @Router /api/data-sources/query/multi [post]
func (c *DataSourceController) QueryMultipleSources(ctx *fiber.Ctx) error {
	var queries []connectors.QueryRequest
	if err := ctx.BodyParser(&queries); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	results, err := c.Service.QueryMultipleSources(ctx.UserContext(), queries)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(results)
}
