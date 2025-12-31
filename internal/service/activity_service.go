package service

import (
	"context"
	"go-crm/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityService interface {
	GetCalendarEvents(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error)
}

type ActivityServiceImpl struct {
	RecordRepo repository.RecordRepository
}

func NewActivityService(recordRepo repository.RecordRepository) ActivityService {
	return &ActivityServiceImpl{RecordRepo: recordRepo}
}

func (s *ActivityServiceImpl) GetCalendarEvents(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	events := []map[string]interface{}{}

	// Fetch Tasks
	tasks, err := s.RecordRepo.List(ctx, "tasks", bson.M{
		"due_date": bson.M{"$gte": start, "$lte": end},
	}, 1000, 0, "due_date", 1)
	if err == nil {
		for _, t := range tasks {
			dueDate := toTime(t["due_date"])
			events = append(events, map[string]interface{}{
				"id":    t["_id"],
				"title": t["subject"],
				"start": dueDate,
				"end":   dueDate,
				"type":  "task",
				"color": "#3b82f6", // Blue
			})
		}
	}

	// Fetch Calls
	calls, err := s.RecordRepo.List(ctx, "calls", bson.M{
		"start_time": bson.M{"$gte": start, "$lte": end},
	}, 1000, 0, "start_time", 1)
	if err == nil {
		for _, c := range calls {
			startT := toTime(c["start_time"])
			// Default duration 30 mins if missing
			duration := 30
			if d, ok := c["duration"].(int32); ok {
				duration = int(d)
			} else if d, ok := c["duration"].(int64); ok {
				duration = int(d)
			} else if d, ok := c["duration"].(float64); ok {
				duration = int(d)
			}

			endT := startT.Add(time.Duration(duration) * time.Minute)

			events = append(events, map[string]interface{}{
				"id":    c["_id"],
				"title": c["subject"],
				"start": startT,
				"end":   endT,
				"type":  "call",
				"color": "#10b981", // Green
			})
		}
	}

	// Fetch Meetings
	meetings, err := s.RecordRepo.List(ctx, "meetings", bson.M{
		"start_time": bson.M{"$gte": start, "$lte": end},
	}, 1000, 0, "start_time", 1)
	if err == nil {
		for _, m := range meetings {
			startT := toTime(m["start_time"])
			endT := toTime(m["end_time"])

			events = append(events, map[string]interface{}{
				"id":    m["_id"],
				"title": m["subject"],
				"start": startT,
				"end":   endT,
				"type":  "meeting",
				"color": "#8b5cf6", // Purple
			})
		}
	}

	return events, nil
}

func toTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	if dt, ok := v.(primitive.DateTime); ok {
		return dt.Time()
	}
	return time.Time{}
}
