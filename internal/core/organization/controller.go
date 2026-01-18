package organization

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/internal/core/audit"
	"go-crm/internal/core/user"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type TenantAuthService interface {
	CreateTenantWithAdmin(ctx context.Context, org *models.Organization, adminEmail, adminPassword, adminFirstName, adminLastName, roleName string) (*models.User, error)
}

type OrganizationController struct {
	service      OrganizationService
	userService  user.UserService
	authService  TenantAuthService
	auditService audit.AuditService
}

func NewOrganizationController(service OrganizationService, userService user.UserService, authService TenantAuthService, auditService audit.AuditService) *OrganizationController {
	return &OrganizationController{
		service:      service,
		userService:  userService,
		authService:  authService,
		auditService: auditService,
	}
}

type CreateTenantWithAdminRequest struct {
	models.Organization `json:",inline"`
	AdminEmail          string `json:"admin_email"`
	AdminPassword       string `json:"admin_password"`
	AdminFirstName      string `json:"admin_first_name"`
	AdminLastName       string `json:"admin_last_name"`
	DefaultRole         string `json:"default_role"`
}

func (c *OrganizationController) CreateOrganization(ctx *fiber.Ctx) error {
	var req CreateTenantWithAdminRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// If admin email is provided, use AuthService for combined creation
	if req.AdminEmail != "" {
		newUser, err := c.authService.CreateTenantWithAdmin(
			ctx.Context(),
			&req.Organization,
			req.AdminEmail,
			req.AdminPassword,
			req.AdminFirstName,
			req.AdminLastName,
			req.DefaultRole,
		)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
			"organization": req.Organization,
			"admin_user":   newUser,
		})
	}

	// Fallback to simple organization creation
	if err := c.service.CreateOrganization(ctx.Context(), &req.Organization); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(req.Organization)
}

func (c *OrganizationController) GetOrganization(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	org, err := c.service.GetOrganization(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(org)
}

func (c *OrganizationController) ListOrganizations(ctx *fiber.Ctx) error {
	// Parse query params for filtering if needed
	// For now, empty filter returns all
	filter := make(map[string]any)

	orgs, err := c.service.ListOrganizations(ctx.Context(), filter)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(orgs)
}

func (c *OrganizationController) UpdateOrganization(ctx *fiber.Ctx) error {
	var org models.Organization
	if err := ctx.BodyParser(&org); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := c.service.UpdateOrganization(ctx.Context(), &org); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(org)
}

func (c *OrganizationController) DeleteOrganization(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.DeleteOrganization(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetOrganizationUsers returns users for a specific organization
// Requires admin permission
func (c *OrganizationController) GetOrganizationUsers(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	page, _ := strconv.ParseInt(ctx.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query("limit", "10"), 10, 64)

	// Create context with target tenant ID
	targetCtx := context.WithValue(ctx.Context(), models.TenantIDKey, id)

	filter := make(map[string]any)
	users, total, err := c.userService.ListUsers(targetCtx, filter, page, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetOrganizationActivity returns audit logs for a specific organization
// Requires admin permission
func (c *OrganizationController) GetOrganizationActivity(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	page, _ := strconv.ParseInt(ctx.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.Query("limit", "20"), 10, 64)

	// Create context with target tenant ID
	targetCtx := context.WithValue(ctx.Context(), models.TenantIDKey, id)

	filters := make(map[string]any)
	logs, err := c.auditService.ListLogs(targetCtx, filters, page, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(logs)
}
