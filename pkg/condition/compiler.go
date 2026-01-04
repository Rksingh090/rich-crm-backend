package condition

import (
	"fmt"
	"strings"
	"time"

	"go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Compiler struct {
	Context map[string]interface{}
}

func NewCompiler(ctx map[string]interface{}) *Compiler {
	return &Compiler{Context: ctx}
}

func (c *Compiler) Compile(group *models.PermissionGroup) (bson.M, error) {
	if group == nil {
		return bson.M{}, nil
	}

	var conditions []bson.M

	// Process Rules
	for _, rule := range group.Rules {
		cond, err := c.compileRule(rule)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
	}

	// Process Nested Groups
	for _, subGroup := range group.Groups {
		cond, err := c.Compile(&subGroup)
		if err != nil {
			return nil, err
		}
		if len(cond) > 0 {
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return bson.M{}, nil
	}

	op := "$and"
	if strings.ToUpper(group.Operator) == "OR" {
		op = "$or"
	}

	return bson.M{op: conditions}, nil
}

func (c *Compiler) compileRule(rule models.PermissionRule) (bson.M, error) {
	val, err := c.resolveValue(rule.Value, rule.Type)
	if err != nil {
		return nil, err
	}

	field := rule.Field

	switch rule.Operator {
	case "eq":
		return bson.M{field: bson.M{"$eq": val}}, nil
	case "ne":
		return bson.M{field: bson.M{"$ne": val}}, nil
	case "gt":
		return bson.M{field: bson.M{"$gt": val}}, nil
	case "lt":
		return bson.M{field: bson.M{"$lt": val}}, nil
	case "gte":
		return bson.M{field: bson.M{"$gte": val}}, nil
	case "lte":
		return bson.M{field: bson.M{"$lte": val}}, nil
	case "in":
		return bson.M{field: bson.M{"$in": val}}, nil
	case "nin":
		return bson.M{field: bson.M{"$nin": val}}, nil
	case "contains":
		if strVal, ok := val.(string); ok {
			return bson.M{field: bson.M{"$regex": primitive.Regex{Pattern: strVal, Options: "i"}}}, nil
		}
		return nil, fmt.Errorf("contains operator requires string value")
	case "startsWith", "starts_with":
		if strVal, ok := val.(string); ok {
			return bson.M{field: bson.M{"$regex": primitive.Regex{Pattern: "^" + strVal, Options: "i"}}}, nil
		}
		return nil, fmt.Errorf("startsWith operator requires string value")
	case "endsWith", "ends_with":
		if strVal, ok := val.(string); ok {
			return bson.M{field: bson.M{"$regex": primitive.Regex{Pattern: strVal + "$", Options: "i"}}}, nil
		}
		return nil, fmt.Errorf("endsWith operator requires string value")
	default:
		return nil, fmt.Errorf("unknown operator: %s", rule.Operator)
	}
}

func (c *Compiler) resolveValue(val interface{}, ruleType models.RuleType) (interface{}, error) {
	if ruleType != models.RuleTypeVariable {
		return val, nil
	}

	// Resolve variable recursively if value is a string starting with $
	strVal, ok := val.(string)
	if !ok {
		return val, nil
	}

	// Simple variable check
	if strings.HasPrefix(strVal, "$") {
		// remove $ for lookup if needed, but Context keys usually include it or we map it
		// standardizing: keys in Context should involve the name, e.g. "user.id"
		key := strings.TrimPrefix(strVal, "$")

		// Special handling for time
		if key == "now" {
			return time.Now(), nil
		}

		if resolved, ok := c.Context[key]; ok {
			return resolved, nil
		}
		return nil, fmt.Errorf("variable not found in context: %s", key)
	}

	return val, nil
}
