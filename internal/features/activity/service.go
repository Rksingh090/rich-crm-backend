package activity

import (
	"context"
	"fmt"
	"go-crm/internal/common/models"
	"go-crm/internal/database"
	"go-crm/internal/features/record"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ActivityService interface {
	GetCalendarEvents(ctx context.Context, start, end time.Time) ([]map[string]any, error)
	GetTimeline(ctx context.Context, moduleName, recordID string) ([]map[string]any, error)
}

type ActivityServiceImpl struct {
	RecordRepo record.RecordRepository
	DB         *database.MongodbDB
}

func NewActivityService(recordRepo record.RecordRepository, db *database.MongodbDB) ActivityService {
	return &ActivityServiceImpl{RecordRepo: recordRepo, DB: db}
}

func (s *ActivityServiceImpl) GetCalendarEvents(ctx context.Context, start, end time.Time) ([]map[string]any, error) {
	events := []map[string]any{}

	// Fetch Tasks
	tasks, err := s.RecordRepo.List(ctx, "tasks", bson.M{
		"due_date": bson.M{"$gte": start, "$lte": end},
	}, nil, 1000, 0, "due_date", 1)
	if err == nil {
		for _, t := range tasks {
			dueDate := toTime(t["due_date"])
			events = append(events, map[string]any{
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
	}, nil, 1000, 0, "start_time", 1)
	if err == nil {
		for _, c := range calls {
			startT := toTime(c["start_time"])
			duration := 30
			if d, ok := c["duration"].(int32); ok {
				duration = int(d)
			} else if d, ok := c["duration"].(int64); ok {
				duration = int(d)
			} else if d, ok := c["duration"].(float64); ok {
				duration = int(d)
			}

			endT := startT.Add(time.Duration(duration) * time.Minute)

			events = append(events, map[string]any{
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
	}, nil, 1000, 0, "start_time", 1)
	if err == nil {
		for _, m := range meetings {
			startT := toTime(m["start_time"])
			endT := toTime(m["end_time"])

			events = append(events, map[string]any{
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

func (s *ActivityServiceImpl) GetTimeline(ctx context.Context, moduleName, recordID string) ([]map[string]any, error) {
	tenantID, ok := ctx.Value(models.TenantIDKey).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("organization context missing")
	}

	db := s.DB.GetTenantDB(tenantID)
	oid, _ := primitive.ObjectIDFromHex(recordID) // Ignore error, check both string and oid

	timeline := []map[string]any{}

	// Collections to check
	// Map Collection Name -> Type Label
	sources := map[string]string{
		"crm_calls":    "call",
		"crm_meetings": "meeting",
		"crm_tasks":    "task",
	}

	// Query Filter: Look for recordID in relationship fields
	// We check both String ID and ObjectID version
	orConditions := []bson.M{
		{"data.related_to": recordID},
		{"data.contact": recordID},
		{"data.account": recordID},
		{"data.lead": recordID},
	}
	if !oid.IsZero() {
		orConditions = append(orConditions,
			bson.M{"data.related_to": oid},
			bson.M{"data.contact": oid},
			bson.M{"data.account": oid},
			bson.M{"data.lead": oid},
		)
	}

	filter := bson.M{
		"__deleted": bson.M{"$ne": true},
		"$or":       orConditions,
	}

	for collName, typeLabel := range sources {
		coll := db.Collection(collName)

		// Optimization: Index on related_to/contact is needed for perf, but strict timeline implies partial scan?
		// We rely on sparse indexes created by EnsureIndexes?
		// Currently unrelated.

		cursor, err := coll.Find(ctx, filter, options.Find().SetLimit(50).SetSort(bson.M{"created_at": -1}))
		if err != nil {
			continue // Skip if collection error or missing
		}

		var results []map[string]any
		if err := cursor.All(ctx, &results); err != nil {
			continue
		}

		for _, item := range results {
			// Extract Data
			data, _ := item["data"].(map[string]any)

			// Determine Date (Priority: start_time > due_date > created_at)
			var date time.Time
			if v, ok := data["start_time"]; ok {
				date = toTime(v)
			} else if v, ok := data["due_date"]; ok {
				date = toTime(v)
			} else {
				date = toTime(item["created_at"])
			}

			// Normalized Activity Item
			timelineItem := map[string]any{
				"id":         item["_id"],
				"type":       typeLabel,
				"module":     collName, // e.g. crm_calls
				"date":       date,
				"subject":    data["subject"], // Assuming subject exists
				"status":     data["status"],
				"owner":      data["owner"], // or assigned_to
				"data":       data,          // full data for UI
				"created_at": item["created_at"],
			}
			timeline = append(timeline, timelineItem)
		}
	}

	// Sort consolidated timeline descending
	sort.Slice(timeline, func(i, j int) bool {
		t1, _ := timeline[i]["date"].(time.Time)
		t2, _ := timeline[j]["date"].(time.Time)
		return t1.After(t2)
	})

	return timeline, nil
}

func toTime(v any) time.Time {
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
