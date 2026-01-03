package audit

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type AuditController struct {
	Service AuditService
}

func NewAuditController(service AuditService) *AuditController {
	return &AuditController{Service: service}
}

// ListLogs godoc
// @Summary List audit logs
// @Description Retrieve a list of audit logs with optional filtering
// @Tags audit
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param module query string false "Filter by module"
// @Param record_id query string false "Filter by record ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/audit/logs [get]
func (ctrl *AuditController) ListLogs(c *fiber.Ctx) error {
	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "20"), 10, 64)

	filters := make(map[string]interface{})
	if module := c.Query("module"); module != "" {
		filters["module"] = module
	}
	if recordID := c.Query("record_id"); recordID != "" {
		filters["record_id"] = recordID
	}

	logs, err := ctrl.Service.ListLogs(c.UserContext(), filters, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(logs)
}
