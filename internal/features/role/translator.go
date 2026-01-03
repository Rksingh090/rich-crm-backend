package role

import (
	"fmt"
	"strings"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TranslateConditions converts a PermissionGroup into a MongoDB filter
func TranslateConditions(group *models.PermissionGroup, contextData map[string]interface{}) (bson.M, error) {
	if group == nil {
		return bson.M{}, nil
	}

	var conditions []bson.M

	// Process Rules
	for _, rule := range group.Rules {
		condition, err := translateRule(rule, contextData)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}

	// Process Nested Groups
	for _, nestedGroup := range group.Groups {
		nestedCondition, err := TranslateConditions(&nestedGroup, contextData)
		if err != nil {
			return nil, err
		}
		if len(nestedCondition) > 0 {
			conditions = append(conditions, nestedCondition)
		}
	}

	if len(conditions) == 0 {
		return bson.M{}, nil
	}

	var operator string
	switch strings.ToUpper(group.Operator) {
	case "OR":
		operator = "$or"
	case "AND":
		operator = "$and"
	default:
		operator = "$and" // Default
	}

	return bson.M{operator: conditions}, nil
}

func translateRule(rule models.PermissionRule, contextData map[string]interface{}) (bson.M, error) {
	val := rule.Value

	// Resolve Variable
	if rule.Type == models.RuleTypeVariable {
		if strVal, ok := val.(string); ok && strings.HasPrefix(strVal, "$") {
			// e.g. $user.id
			resolved, ok := resolvePath(strVal, contextData)
			if !ok {
				// If variable not found, what to do? Fail safe: strict deny or null?
				// Let's assume null for now, or error.
				return nil, fmt.Errorf("variable not found: %s", strVal)
			}
			val = resolved
		}
	}

	// Construct Mongo Query
	// Data fields are stored in "data.field" in the entity_records collection.
	// But the user enters "status" or "owner_id".
	// We should map standard fields (created_by, etc) differently from dynamic data fields.
	// Standard System Fields: _id, created_at, updated_at, created_by, updated_by, tenant_id, product, entity
	// All else inside "data.".

	dbField := mapFieldToDB(rule.Field)

	switch rule.Operator {
	case "eq":
		return bson.M{dbField: bson.M{"$eq": val}}, nil
	case "ne":
		return bson.M{dbField: bson.M{"$ne": val}}, nil
	case "gt":
		return bson.M{dbField: bson.M{"$gt": val}}, nil
	case "gte":
		return bson.M{dbField: bson.M{"$gte": val}}, nil
	case "lt":
		return bson.M{dbField: bson.M{"$lt": val}}, nil
	case "lte":
		return bson.M{dbField: bson.M{"$lte": val}}, nil
	case "in":
		return bson.M{dbField: bson.M{"$in": val}}, nil
	case "nin":
		return bson.M{dbField: bson.M{"$nin": val}}, nil
	case "contains":
		// Regex case insensitive
		return bson.M{dbField: bson.M{"$regex": val, "$options": "i"}}, nil
	default:
		return bson.M{dbField: bson.M{"$eq": val}}, nil
	}
}

func mapFieldToDB(field string) string {
	// Map known system fields to top level
	switch field {
	case "id", "_id":
		return "_id"
	case "created_by":
		return "created_by"
	case "updated_by":
		return "updated_by"
	case "created_at":
		return "created_at"
	case "updated_at":
		return "updated_at"
	case "tenant_id":
		return "tenant_id"
	// Common CRM fields that are typically stored in data
	case "owner", "owner_id", "ownerId":
		return "data.owner"
	case "ownerGroup", "owner_group":
		return "data.ownerGroup"
	case "status":
		return "data.status"
	case "assignedTo", "assigned_to":
		return "data.assignedTo"
	default:
		// All other fields are assumed to be in the data object
		return "data." + field
	}
}

func resolvePath(path string, data map[string]interface{}) (interface{}, bool) {
	// Simple resolution for now: $user.id, $user.org_id
	// path starts with $, strip it
	key := strings.TrimPrefix(path, "$")

	val, ok := data[key]
	// Handle ObjectID conversion if needed?
	// The contextData should already have correct types (ObjectIDs etc)
	return val, ok
}

// PrepareContextData prepares standard variables for ABAC condition evaluation
// This includes user ID, organization ID, and user groups
func PrepareContextData(userID primitive.ObjectID, orgID primitive.ObjectID, groups []string) map[string]interface{} {
	return map[string]interface{}{
		"user.id":     userID.Hex(), // Store as string to match stored IDs in EntityRecord
		"user.org_id": orgID.Hex(),
		"user.groups": groups, // Array of group names for "in" operator
	}
}
