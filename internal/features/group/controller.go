package group

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GroupController struct {
	Service GroupService
}

func NewGroupController(service GroupService) *GroupController {
	return &GroupController{Service: service}
}

// CreateGroup godoc
// CreateGroup godoc
// @Summary Create group
// @Description Create a new user group
// @Tags groups
// @Accept json
// @Produce json
// @Param group body Group true "Group Details"
// @Success 201 {object} Group
// @Failure 400 {object} map[string]interface{}
// @Router /api/groups [post]
func (c *GroupController) CreateGroup(ctx *fiber.Ctx) error {
	var group Group
	if err := ctx.BodyParser(&group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := c.Service.CreateGroup(ctx.UserContext(), &group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(group)
}

// GetAllGroups godoc
// GetAllGroups godoc
// @Summary List groups
// @Description List all user groups
// @Tags groups
// @Produce json
// @Success 200 {array} Group
// @Failure 500 {object} map[string]interface{}
// @Router /api/groups [get]
func (c *GroupController) GetAllGroups(ctx *fiber.Ctx) error {
	groups, err := c.Service.GetAllGroups(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(groups)
}

// GetGroup godoc
// GetGroup godoc
// @Summary Get group
// @Description Get a group by ID
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} Group
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/groups/{id} [get]
func (c *GroupController) GetGroup(ctx *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	group, err := c.Service.GetGroupByID(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	return ctx.JSON(group)
}

// UpdateGroup godoc
// UpdateGroup godoc
// @Summary Update group
// @Description Update an existing group
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param group body Group true "Group Details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/groups/{id} [put]
func (c *GroupController) UpdateGroup(ctx *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	var group Group
	if err := ctx.BodyParser(&group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := c.Service.UpdateGroup(ctx.UserContext(), id, &group); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Group updated successfully",
	})
}

// DeleteGroup godoc
// DeleteGroup godoc
// @Summary Delete group
// @Description Delete a group by ID
// @Tags groups
// @Param id path string true "Group ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/groups/{id} [delete]
func (c *GroupController) DeleteGroup(ctx *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	if err := c.Service.DeleteGroup(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Group deleted successfully",
	})
}

// AddMember godoc
// AddMember godoc
// @Summary Add group member
// @Description Add a user to a group
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param body body map[string]string true "User ID Object {user_id: ...}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/groups/{id}/members [post]
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

	if err := c.Service.AddMember(ctx.UserContext(), groupID, userID); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Member added successfully",
	})
}

// RemoveMember godoc
// RemoveMember godoc
// @Summary Remove group member
// @Description Remove a user from a group
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/groups/{id}/members/{user_id} [delete]
func (c *GroupController) RemoveMember(ctx *fiber.Ctx) error {
	groupID, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	userID, err := primitive.ObjectIDFromHex(ctx.Params("user_id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := c.Service.RemoveMember(ctx.UserContext(), groupID, userID); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Member removed successfully",
	})
}
