package auth

import (
	"github.com/gofiber/fiber/v2"
)

type AuthController struct {
	AuthService AuthService
}

func NewAuthController(authService AuthService) *AuthController {
	return &AuthController{
		AuthService: authService,
	}
}

type RegisterRequest struct {
	Password string   `json:"password"`
	Email    string   `json:"email"`
	OrgName  string   `json:"org_name"`
	Plan     string   `json:"plan"`
	Apps     []string `json:"apps"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user with password and email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body RegisterRequest true "Register Input"
// @Success      201  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to create user"
// @Router       /register [post]
func (ctrl *AuthController) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	_, err := ctrl.AuthService.Register(c.Context(), req.Password, req.Email, req.OrgName, req.Plan, req.Apps)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
	})
}

// Login godoc
// @Summary      Login
// @Description  Login with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body LoginRequest true "Login Input"
// @Success      200  {object} AuthResponse
// @Failure      400  {string} string "Invalid request body"
// @Failure      401  {string} string "Invalid credentials"
// @Router       /api/login [post]
func (ctrl *AuthController) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	token, err := ctrl.AuthService.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(AuthResponse{Token: token})
}

type CreateTenantAdminRequest struct {
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (ctrl *AuthController) CreateTenantAdmin(c *fiber.Ctx) error {
	var req CreateTenantAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.TenantID == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "tenant_id, email, and password are required",
		})
	}

	user, err := ctrl.AuthService.CreateTenantAdmin(c.Context(), req.TenantID, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Tenant Admin created successfully",
		"user_id": user.ID,
	})
}
