package controllers

import (
	"strconv"

	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecordController struct {
	Service service.RecordService
}

func NewRecordController(service service.RecordService) *RecordController {
	return &RecordController{
		Service: service,
	}
}

// CreateRecord godoc
// @Summary      Create a record in a module
// @Description  Insert data into a dynamic module with validation
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        input body map[string]interface{} true "Record Data"
// @Success      201  {object} map[string]interface{}
// @Failure      400  {string} string "Invalid request"
// @Failure      404  {string} string "Module not found"
// @Router       /api/modules/{name}/records [post]
func (ctrl *RecordController) CreateRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from context
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	id, err := ctrl.Service.CreateRecord(c.Context(), moduleName, data, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Record created successfully",
		"id":      id,
	})
}

// ListRecords godoc
// @Summary      List records from a module
// @Description  Get records with pagination and filters
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name  path string true "Module Name"
// @Param        page  query int    false "Page number (default 1)"
// @Param        limit query int    false "Records per page (default 10)"
// @Success      200   {array}  map[string]interface{}
// @Failure      400   {string} string "Invalid input"
// @Router       /api/modules/{name}/records [get]
func (ctrl *RecordController) ListRecords(c *fiber.Ctx) error {
	moduleName := c.Params("name")

	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)
	sortBy := c.Query("sort_by", "created_at")
	sortOrder := c.Query("order", "desc")

	// Extract filters from query params
	filters := make(map[string]interface{})
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if k != "page" && k != "limit" && k != "sort_by" && k != "order" {
			filters[k] = string(value)
		}
	})

	// Get user ID from context
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	records, totalCount, err := ctrl.Service.ListRecords(c.Context(), moduleName, filters, page, limit, sortBy, sortOrder, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": records,
		"meta": fiber.Map{
			"total": totalCount,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetRecord godoc
// @Summary      Get a single record
// @Description  Get a record by ID
// @Tags         records
// @Produce      json
// @Param        module path string true "Module Name"
// @Param        id     path string true "Record ID"
// @Success      200    {object} map[string]any
// @Failure      404    {string} string "Not Found"
// @Router       /api/modules/{module}/records/{id} [get]
func (ctrl *RecordController) GetRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	// Get user ID from context
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	record, err := ctrl.Service.GetRecord(c.Context(), moduleName, id, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(record)
}

// UpdateRecord godoc
// @Summary      Update a record
// @Description  Update a record in a module
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        id   path string true "Record ID"
// @Param        input body map[string]interface{} true "Record Data"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/modules/{name}/records/{id} [put]
func (ctrl *RecordController) UpdateRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from context
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := ctrl.Service.UpdateRecord(c.Context(), moduleName, id, data, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Record updated successfully",
	})
}

// DeleteRecord godoc
// @Summary      Delete a record
// @Description  Delete a record from a module
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        name path string true "Module Name"
// @Param        id   path string true "Record ID"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid input"
// @Router       /api/modules/{name}/records/{id} [delete]
func (ctrl *RecordController) DeleteRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	if err := ctrl.Service.DeleteRecord(c.Context(), moduleName, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Record deleted successfully",
	})
}
