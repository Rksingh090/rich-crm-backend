package service

import (
	"context"
	"go-crm/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
			events = append(events, map[string]interface{}{
				"id":    t["_id"],
				"title": t["subject"],
				"start": t["due_date"],
				"end":   t["due_date"],
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
			startT := c["start_time"].(time.Time)
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
			var startT, endT time.Time
			if t, ok := m["start_time"].(time.Time); ok {
				startT = t
			}
			if t, ok := m["end_time"].(time.Time); ok {
				endT = t
			}

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
