package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-crm/internal/service"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	UserService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
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
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit < 1 {
		limit = 10
	}

	filter := make(map[string]interface{})
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	users, total, err := h.UserService.ListUsers(r.Context(), filter, page, limit)
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.UserService.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
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
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
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

	if err := h.UserService.UpdateUser(r.Context(), id, updates); err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
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
func (h *UserHandler) UpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.UserService.UpdateUserRoles(r.Context(), id, req.RoleIDs); err != nil {
		http.Error(w, "Failed to update user roles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User roles updated successfully"})
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
func (h *UserHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.UserService.UpdateUserStatus(r.Context(), id, req.Status); err != nil {
		http.Error(w, "Failed to update user status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User status updated successfully"})
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
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.UserService.DeleteUser(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}
