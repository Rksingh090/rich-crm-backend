package controllers

import (
	"go-crm/internal/models"
	"go-crm/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GroupController struct {
	service *service.GroupService
}

func NewGroupController(service *service.GroupService) *GroupController {
	return &GroupController{service: service}
}

// CreateGroup godoc
// @Summary      Create a new group
// @Description  Create a new group with permissions
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        group body models.Group true "Group object"
// @Success      201  {object} models.Group
// @Failure      400  {object} map[string]string
// @Router       /api/groups [post]
func (c *GroupController) CreateGroup(ctx *fiber.Ctx) error {
	var group models.Group
	if err := ctx.BodyParser(&group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := c.service.CreateGroup(ctx.Context(), &group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(group)
}

// GetAllGroups godoc
// @Summary      Get all groups
// @Description  Retrieve all groups
// @Tags         groups
// @Produce      json
// @Success      200  {array} models.Group
// @Failure      500  {object} map[string]string
// @Router       /api/groups [get]
func (c *GroupController) GetAllGroups(ctx *fiber.Ctx) error {
	groups, err := c.service.GetAllGroups(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(groups)
}

// GetGroup godoc
// @Summary      Get a group by ID
// @Description  Retrieve a specific group
// @Tags         groups
// @Produce      json
// @Param        id path string true "Group ID"
// @Success      200  {object} models.Group
// @Failure      404  {object} map[string]string
// @Router       /api/groups/{id} [get]
func (c *GroupController) GetGroup(ctx *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	group, err := c.service.GetGroupByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	return ctx.JSON(group)
}

// UpdateGroup godoc
// @Summary      Update a group
// @Description  Update group details and permissions
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Param        group body models.Group true "Group object"
// @Success      200  {object} map[string]string
// @Failure      400  {object} map[string]string
// @Router       /api/groups/{id} [put]
func (c *GroupController) UpdateGroup(ctx *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	var group models.Group
	if err := ctx.BodyParser(&group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := c.service.UpdateGroup(ctx.Context(), id, &group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Group updated successfully",
	})
}

// DeleteGroup godoc
// @Summary      Delete a group
// @Description  Delete a group by ID
// @Tags         groups
// @Produce      json
// @Param        id path string true "Group ID"
// @Success      200  {object} map[string]string
// @Failure      400  {object} map[string]string
// @Router       /api/groups/{id} [delete]
func (c *GroupController) DeleteGroup(ctx *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	if err := c.service.DeleteGroup(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Group deleted successfully",
	})
}

// AddMember godoc
// @Summary      Add member to group
// @Description  Add a user to a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Param        body body map[string]string true "User ID"
// @Success      200  {object} map[string]string
// @Failure      400  {object} map[string]string
// @Router       /api/groups/{id}/members [post]
func (c *GroupController) AddMember(ctx *fiber.Ctx) error {
	groupID, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID, err := primitive.ObjectIDFromHex(body.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := c.service.AddMember(ctx.Context(), groupID, userID); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Member added successfully",
	})
}

// RemoveMember godoc
// @Summary      Remove member from group
// @Description  Remove a user from a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path string true "Group ID"
// @Param        userId path string true "User ID"
// @Success      200  {object} map[string]string
// @Failure      400  {object} map[string]string
// @Router       /api/groups/{id}/members/{userId} [delete]
func (c *GroupController) RemoveMember(ctx *fiber.Ctx) error {
	groupID, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	userID, err := primitive.ObjectIDFromHex(ctx.Params("userId"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := c.service.RemoveMember(ctx.Context(), groupID, userID); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Member removed successfully",
	})
}
