package controllers

import (
	"strconv"

	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AuditController struct {
	Service service.AuditService
}

func NewAuditController(service service.AuditService) *AuditController {
	return &AuditController{Service: service}
}

// ListLogs godoc
// @Summary      List audit logs
// @Description  Get audit logs with pagination
// @Tags         audit
// @Accept       json
// @Produce      json
// @Param        page  query int    false "Page number (default 1)"
// @Param        limit query int    false "Logs per page (default 20)"
// @Success      200   {array}  models.AuditLog
// @Failure      500   {string} string "Internal Server Error"
// @Router       /audit-logs [get]
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

	logs, err := ctrl.Service.ListLogs(c.Context(), filters, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(logs)
}
