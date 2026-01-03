package saved_filter

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SavedFilterService interface {
	CreateFilter(ctx context.Context, filter *SavedFilter) error
	GetFilter(ctx context.Context, id string) (*SavedFilter, error)
	UpdateFilter(ctx context.Context, filter *SavedFilter) error
	DeleteFilter(ctx context.Context, id string, userID primitive.ObjectID) error
	GetUserFilters(ctx context.Context, userID primitive.ObjectID, moduleName string) ([]SavedFilter, error)
	GetPublicFilters(ctx context.Context, moduleName string) ([]SavedFilter, error)
	BuildQueryFromCriteria(criteria FilterCriteria) map[string]interface{}
}

type SavedFilterServiceImpl struct {
	FilterRepo SavedFilterRepository
}

func NewSavedFilterService(filterRepo SavedFilterRepository) SavedFilterService {
	return &SavedFilterServiceImpl{
		FilterRepo: filterRepo,
	}
}

func (s *SavedFilterServiceImpl) CreateFilter(ctx context.Context, filter *SavedFilter) error {
	return s.FilterRepo.Create(ctx, filter)
}

func (s *SavedFilterServiceImpl) GetFilter(ctx context.Context, id string) (*SavedFilter, error) {
	return s.FilterRepo.Get(ctx, id)
}

func (s *SavedFilterServiceImpl) UpdateFilter(ctx context.Context, filter *SavedFilter) error {
	return s.FilterRepo.Update(ctx, filter)
}

func (s *SavedFilterServiceImpl) DeleteFilter(ctx context.Context, id string, userID primitive.ObjectID) error {
	filter, err := s.FilterRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if filter.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	return s.FilterRepo.Delete(ctx, id)
}

func (s *SavedFilterServiceImpl) GetUserFilters(ctx context.Context, userID primitive.ObjectID, moduleName string) ([]SavedFilter, error) {
	return s.FilterRepo.FindByUser(ctx, userID.Hex(), moduleName)
}

func (s *SavedFilterServiceImpl) GetPublicFilters(ctx context.Context, moduleName string) ([]SavedFilter, error) {
	return s.FilterRepo.FindPublic(ctx, moduleName)
}

func (s *SavedFilterServiceImpl) BuildQueryFromCriteria(criteria FilterCriteria) map[string]interface{} {
	query := make(map[string]interface{})

	if len(criteria.Conditions) == 0 && len(criteria.Groups) == 0 {
		return query
	}

	var conditions []map[string]interface{}

	for _, condition := range criteria.Conditions {
		condQuery := s.buildConditionQuery(condition)
		if condQuery != nil {
			conditions = append(conditions, condQuery)
		}
	}

	for _, group := range criteria.Groups {
		groupQuery := s.BuildQueryFromCriteria(group)
		if len(groupQuery) > 0 {
			conditions = append(conditions, groupQuery)
		}
	}

	if len(conditions) == 0 {
		return query
	}

	if criteria.Logic == "OR" {
		query["$or"] = conditions
	} else {
		query["$and"] = conditions
	}

	return query
}

func (s *SavedFilterServiceImpl) buildConditionQuery(condition FilterCondition) map[string]interface{} {
	query := make(map[string]interface{})

	switch condition.Operator {
	case "eq":
		query[condition.Field] = condition.Value
	case "ne":
		query[condition.Field] = map[string]interface{}{"$ne": condition.Value}
	case "gt":
		query[condition.Field] = map[string]interface{}{"$gt": condition.Value}
	case "gte":
		query[condition.Field] = map[string]interface{}{"$gte": condition.Value}
	case "lt":
		query[condition.Field] = map[string]interface{}{"$lt": condition.Value}
	case "lte":
		query[condition.Field] = map[string]interface{}{"$lte": condition.Value}
	case "contains":
		query[condition.Field] = map[string]interface{}{"$regex": condition.Value, "$options": "i"}
	case "in":
		query[condition.Field] = map[string]interface{}{"$in": condition.Value}
	case "nin":
		query[condition.Field] = map[string]interface{}{"$nin": condition.Value}
	default:
		query[condition.Field] = condition.Value
	}

	return query
}
