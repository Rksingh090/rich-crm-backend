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
func (ctrl *RecordController) CreateRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("userID").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	res, err := ctrl.Service.CreateRecord(c.Context(), moduleName, data, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// GetRecord godoc
func (ctrl *RecordController) GetRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("userID").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	record, err := ctrl.Service.GetRecord(c.Context(), moduleName, id, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Record not found",
		})
	}

	return c.JSON(record)
}

// ListRecords godoc
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
	if idStr, ok := c.Locals("userID").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	records, total, err := ctrl.Service.ListRecords(c.Context(), moduleName, filters, page, limit, sortBy, sortOrder, userID)
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
	if idStr, ok := c.Locals("userID").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
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
func (ctrl *RecordController) DeleteRecord(c *fiber.Ctx) error {
	moduleName := c.Params("name")
	id := c.Params("id")

	if err := ctrl.Service.DeleteRecord(c.Context(), moduleName, id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Record deleted successfully",
	})
}
