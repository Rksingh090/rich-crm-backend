package audit

import (
	"context"
	"strconv"

	"go-crm/internal/common/models"
	"go-crm/internal/middleware"
	"go-crm/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gofiber/fiber/v2"
)

type AuditController struct {
	Service     AuditService
	RoleService middleware.RoleService
}

func NewAuditController(service AuditService, roleService middleware.RoleService) *AuditController {
	return &AuditController{
		Service:     service,
		RoleService: roleService,
	}
}

// ListLogs godoc
// @Summary List audit logs
// @Description Retrieve a list of audit logs with optional filtering. Requires 'crm.settings_audit_logs' permission OR 'read' permission for the specific module being filtered.
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

	// Permission Check
	// 1. If filtering by module, check if user has access to that module
	// 2. Otherwise/If failed, check for global audit log permission

	// Get user claims from context
	claimsInterface := c.Locals(utils.UserClaimsKey)
	if claimsInterface == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	claims, ok := claimsInterface.(*utils.UserClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authentication claims",
		})
	}

	hasAccess := false
	moduleName := c.Query("module")

	if moduleName != "" && len(claims.Roles) > 0 {
		// Try with app prefix (e.g., "crm.leads")
		resourceName := "crm." + moduleName
		hasPermission, err := ctrl.RoleService.CheckModulePermission(c.UserContext(), claims.Roles, resourceName, "read")
		if err == nil && hasPermission {
			hasAccess = true
		}
	}

	if !hasAccess && len(claims.Roles) > 0 {
		hasPermission, err := ctrl.RoleService.CheckModulePermission(c.UserContext(), claims.Roles, "crm.settings_audit_logs", "read")
		if err == nil && hasPermission {
			hasAccess = true
		}
	}

	if !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: Insufficient permissions for this action",
		})
	}

	logs, err := ctrl.Service.ListLogs(c.UserContext(), filters, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(logs)
}

// GetGlobalActivity returns audit logs across all tenants
// Requires global admin permission
func (ctrl *AuditController) GetGlobalActivity(c *fiber.Ctx) error {
	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "50"), 10, 64)

	filters := make(map[string]interface{})
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		if oid, err := primitive.ObjectIDFromHex(tenantID); err == nil {
			filters["tenant_id"] = oid
		}
	}

	// Create a context that definitely does NOT have a tenant_id constraint
	// We pass an empty string to override any existing value
	globalCtx := context.WithValue(c.UserContext(), models.TenantIDKey, "")

	logs, err := ctrl.Service.ListLogs(globalCtx, filters, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(logs)
}
