package record

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecordController struct {
	Service RecordService
}

func NewRecordController(service RecordService) *RecordController {
	return &RecordController{Service: service}
}

// CreateRecord godoc
// CreateRecord godoc
// @Summary Create record
// @Description Create a new record in a module
// @Tags records
// @Accept json
// @Produce json
// @Param name path string true "Module Name"
// @Param record body map[string]interface{} true "Record Data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/records/{name} [post]
func (ctrl *RecordController) CreateRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	res, err := ctrl.Service.CreateRecord(c.UserContext(), moduleName, data, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// GetRecord godoc
// GetRecord godoc
// @Summary Get record
// @Description Get a record by ID
// @Tags records
// @Produce json
// @Param name path string true "Module Name"
// @Param id path string true "Record ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/records/{name}/{id} [get]
func (ctrl *RecordController) GetRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	record, err := ctrl.Service.GetRecord(c.UserContext(), moduleName, id, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Record not found",
		})
	}

	return c.JSON(record)
}

// ListRecords godoc
// ListRecords godoc
// @Summary List records
// @Description List records in a module with filtering, sorting, and pagination
// @Tags records
// @Produce json
// @Param name path string true "Module Name"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param sort_by query string false "Sort field"
// @Param sort_order query string false "Sort order (asc/desc)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/records/{name} [get]
func (ctrl *RecordController) ListRecords(c *fiber.Ctx) error {
	moduleName := c.Params("name")

	// Pagination
	page := ParseInt64(c.Query("page", "1"), 1)
	limit := ParseInt64(c.Query("limit", "10"), 10)

	// Sorting
	sortBy := c.Query("sort_by", "created_at")
	sortOrder := c.Query("sort_order", "desc")

	// Filtering
	filters := make(map[string]interface{})
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if k != "page" && k != "limit" && k != "sort_by" && k != "sort_order" {
			filters[k] = string(value)
		}
	})

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	records, total, err := ctrl.Service.ListRecords(c.UserContext(), moduleName, filters, page, limit, sortBy, sortOrder, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":  records,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// UpdateRecord godoc
// UpdateRecord godoc
// @Summary Update record
// @Description Update an existing record
// @Tags records
// @Accept json
// @Produce json
// @Param name path string true "Module Name"
// @Param id path string true "Record ID"
// @Param record body map[string]interface{} true "Record Data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/records/{name}/{id} [put]
func (ctrl *RecordController) UpdateRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := ctrl.Service.UpdateRecord(c.UserContext(), moduleName, id, data, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Record updated successfully",
	})
}

// DeleteRecord godoc
// DeleteRecord godoc
// @Summary Delete record
// @Description Delete a record by ID
// @Tags records
// @Param name path string true "Module Name"
// @Param id path string true "Record ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/records/{name}/{id} [delete]
func (ctrl *RecordController) DeleteRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	if err := ctrl.Service.DeleteRecord(c.UserContext(), moduleName, id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Record deleted successfully",
	})
}
