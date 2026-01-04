package record

import (
	"encoding/json"
	"strings"

	common_models "go-crm/internal/common/models"

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
	sortOrder := c.Query("sort_order")
	if sortOrder == "" {
		sortOrder = c.Query("order", "desc")
	}

	// Filtering
	// Filtering
	var filters []common_models.Filter

	// Check if "filters" query param exists (JSON encoded)
	if filtersStr := c.Query("filters"); filtersStr != "" {
		_ = json.Unmarshal([]byte(filtersStr), &filters)
	}

	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if k != "page" && k != "limit" && k != "sort_by" && k != "sort_order" && k != "order" && k != "filters" {
			v := string(value)
			// Parse field__operator
			fieldName := k
			operator := "eq"
			if strings.Contains(k, "__") {
				parts := strings.Split(k, "__")
				if len(parts) == 2 {
					fieldName = parts[0]
					operator = parts[1]
				}
			}

			// Handle comma-separated values for 'in' operator
			var typeVal interface{} = v
			if operator == "in" || operator == "nin" {
				typeVal = strings.Split(v, ",")
			}

			filters = append(filters, common_models.Filter{
				Field:    fieldName,
				Operator: operator,
				Value:    typeVal,
			})
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

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	if err := ctrl.Service.DeleteRecord(c.UserContext(), moduleName, id, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Record deleted successfully",
	})
}

// QueryRecords godoc
// @Summary Query records with strict permission checks
// @Description Query records based on resource, action, and filters
// @Tags records
// @Accept json
// @Produce json
// @Param query body map[string]interface{} true "Query Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /api/records/query [post]
func (ctrl *RecordController) QueryRecords(c *fiber.Ctx) error {
	var req struct {
		Resource  string                 `json:"resource"`
		Action    string                 `json:"action"`
		Filters   map[string]interface{} `json:"filters"`
		Page      int64                  `json:"page"`
		Limit     int64                  `json:"limit"`
		SortBy    string                 `json:"sort_by"`
		SortOrder string                 `json:"sort_order"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.Resource == "" || req.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "resource and action are required"})
	}

	// Pagination defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert map filters to []common_models.Filter
	// Logic: Key matches "field" or "field__operator". Value matches value.
	// Defaults to eq if no operator in key.
	var filters []common_models.Filter
	for k, v := range req.Filters {
		field := k
		operator := "eq"
		if strings.Contains(k, "__") {
			parts := strings.Split(k, "__")
			if len(parts) == 2 {
				field = parts[0]
				operator = parts[1]
			}
		}

		// Handle comma-separated strings for 'in', 'between' etc if needed?
		// Or assume JSON arrays for 'in'.
		// Map conversion logic from ListRecords handles comma-separated strings for query params.
		// For JSON body, array is preferred.
		// But existing logic in ListRecords handles string split.
		// Here we take raw interface{}.

		filters = append(filters, common_models.Filter{
			Field:    field,
			Operator: operator,
			Value:    v,
		})
	}

	var userID primitive.ObjectID
	if idStr, ok := c.Locals("user_id").(string); ok && idStr != "" {
		userID, _ = primitive.ObjectIDFromHex(idStr)
	}

	records, total, err := ctrl.Service.QueryRecords(c.UserContext(), req.Resource, req.Action, filters, req.Page, req.Limit, req.SortBy, req.SortOrder, userID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "not allowed") {
			status = fiber.StatusForbidden
		} else if strings.Contains(err.Error(), "not found") {
			status = fiber.StatusNotFound
		} else if strings.Contains(err.Error(), "invalid") {
			status = fiber.StatusBadRequest
		}

		return c.Status(status).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":  records,
		"total": total,
		"page":  req.Page,
		"limit": req.Limit,
	})
}
