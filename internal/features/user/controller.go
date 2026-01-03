package user

import (
	"strconv"

	"go-crm/internal/common/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserController struct {
	UserService UserService
}

func NewUserController(userService UserService) *UserController {
	return &UserController{
		UserService: userService,
	}
}

type UpdateUserRequest struct {
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status,omitempty"`
	ReportsTo string `json:"reports_to,omitempty"`
}

type CreateUserRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status,omitempty"`
	ReportsTo string `json:"reports_to,omitempty"`
}

type UpdateUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// ListUsers godoc
// @Summary      List all users
// @Description  Get a paginated list of users
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(10)
// @Param        status query string false "Filter by status"
// @Success      200  {object} map[string]interface{}
// @Failure      500  {string} string "Failed to fetch users"
// @Router       /users [get]
func (ctrl *UserController) ListUsers(c *fiber.Ctx) error {
	page, _ := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)

	filter := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	users, total, err := ctrl.UserService.ListUsers(c.Context(), filter, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Get detailed information about a specific user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200  {object} models.User
// @Failure      404  {string} string "User not found"
// @Router       /users/{id} [get]
func (ctrl *UserController) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")

	user, err := ctrl.UserService.GetUserByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

// CreateUser godoc
// @Summary      Create user
// @Description  Create a new user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        input body CreateUserRequest true "Create User Input"
// @Success      201  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to create user"
// @Router       /users [post]
func (ctrl *UserController) CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username, email and password are required",
		})
	}

	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password, // TODO: Use bcrypt hashing
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Status:    req.Status,
	}

	if req.ReportsTo != "" {
		if oid, err := primitive.ObjectIDFromHex(req.ReportsTo); err == nil {
			user.ReportsTo = &oid
		}
	}

	if err := ctrl.UserService.CreateUser(c.Context(), user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"id":      user.ID.Hex(),
	})
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Update user information
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Param        input body UpdateUserRequest true "Update User Input"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to update user"
// @Router       /users/{id} [put]
func (ctrl *UserController) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.ReportsTo != "" {
		if req.ReportsTo == "null" {
			updates["reports_to"] = nil
		} else if oid, err := primitive.ObjectIDFromHex(req.ReportsTo); err == nil {
			updates["reports_to"] = oid
		}
	}

	if err := ctrl.UserService.UpdateUser(c.Context(), id, updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User updated successfully",
	})
}

// UpdateUserRoles godoc
// @Summary      Update user roles
// @Description  Update the roles assigned to a user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Param        input body UpdateUserRolesRequest true "Update User Roles Input"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to update user roles"
// @Router       /users/{id}/roles [put]
func (ctrl *UserController) UpdateUserRoles(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateUserRolesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.UserService.UpdateUserRoles(c.Context(), id, req.RoleIDs); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user roles: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User roles updated successfully",
	})
}

// UpdateUserStatus godoc
// @Summary      Update user status
// @Description  Update the status of a user (active, inactive, suspended)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Param        input body UpdateUserStatusRequest true "Update User Status Input"
// @Success      200  {object} map[string]string
// @Failure      400  {string} string "Invalid request body"
// @Failure      500  {string} string "Failed to update user status"
// @Router       /users/{id}/status [put]
func (ctrl *UserController) UpdateUserStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateUserStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := ctrl.UserService.UpdateUserStatus(c.Context(), id, req.Status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user status: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User status updated successfully",
	})
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Delete a user by ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200  {object} map[string]string
// @Failure      500  {string} string "Failed to delete user"
// @Router       /users/{id} [delete]
func (ctrl *UserController) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.UserService.DeleteUser(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
