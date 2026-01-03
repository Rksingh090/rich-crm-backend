package search

import (
	"context"
	"fmt"
	"strings"

	"go-crm/internal/database"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SearchResult struct {
	Type        string `json:"type"` // "module", "record", "page", "user"
	Title       string `json:"title"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Icon        string `json:"icon,omitempty"`
}

type SearchService interface {
	GlobalSearch(ctx context.Context, query string, userID primitive.ObjectID) ([]SearchResult, error)
}

type SearchServiceImpl struct {
	db            *database.MongodbDB
	moduleService module.ModuleService
	recordService record.RecordService
}

func NewSearchService(db *database.MongodbDB, moduleService module.ModuleService, recordService record.RecordService) SearchService {
	return &SearchServiceImpl{
		db:            db,
		moduleService: moduleService,
		recordService: recordService,
	}
}

func (s *SearchServiceImpl) GlobalSearch(ctx context.Context, query string, userID primitive.ObjectID) ([]SearchResult, error) {
	var results []SearchResult
	query = strings.TrimSpace(query)
	if query == "" {
		return results, nil
	}

	// 1. Search Modules
	modules, err := s.moduleService.ListModules(ctx, userID)
	if err == nil {
		for _, m := range modules {
			if strings.Contains(strings.ToLower(m.Name), strings.ToLower(query)) || strings.Contains(strings.ToLower(m.Label), strings.ToLower(query)) {
				results = append(results, SearchResult{
					Type:        "module",
					Title:       m.Label,
					Name:        m.Name,
					Description: fmt.Sprintf("Go to %s module", m.Label),
					Link:        fmt.Sprintf("/dashboard/modules/%s", m.Name),
					Icon:        "box",
				})
			}
		}
	}

	// 2. Search Static Pages (Settings)
	staticPages := []SearchResult{
		{Type: "page", Title: "Overview", Description: "Overview", Link: "/dashboard", Icon: "layout-dashboard"},
		{Type: "page", Title: "Tickets", Description: "Tickets", Link: "/dashboard/tickets", Icon: "ticket"},
		{Type: "page", Title: "Reports", Description: "Reports", Link: "/dashboard/reports", Icon: "file-text"},
		{Type: "page", Title: "General", Description: "General Settings", Link: "/dashboard/settings", Icon: "settings"},
		{Type: "page", Title: "Email Configuration", Description: "Email Configuration", Link: "/dashboard/settings/email", Icon: "mail"},
		{Type: "page", Title: "Module Builder", Description: "Module Builder", Link: "/dashboard/settings/modules", Icon: "layers"},
		{Type: "page", Title: "Audit Logs", Description: "Audit Logs", Link: "/dashboard/settings/audit-logs", Icon: "file-text"},
		{Type: "page", Title: "User Management", Description: "User Management", Link: "/dashboard/settings/users", Icon: "users"},
		{Type: "page", Title: "Roles & Permissions", Description: "Roles & Permissions", Link: "/dashboard/settings/roles", Icon: "shield"},
		{Type: "page", Title: "Groups", Description: "Groups", Link: "/dashboard/settings/groups", Icon: "users"},
		{Type: "page", Title: "Automation", Description: "Automation", Link: "/dashboard/settings/automation", Icon: "workflow"},
		{Type: "page", Title: "Workflow Automation", Description: "Workflow Automation", Link: "/dashboard/settings/workflows", Icon: "workflow"},
		{Type: "page", Title: "SLA Policies", Description: "SLA Policies", Link: "/dashboard/settings/sla-policies", Icon: "clock"},
		{Type: "page", Title: "Escalation Rules", Description: "Escalation Rules", Link: "/dashboard/settings/escalation-rules", Icon: "alert-triangle"},
		{Type: "page", Title: "Integration", Description: "Integration", Link: "/dashboard/settings/integration", Icon: "integration"},
		{Type: "page", Title: "Webhooks", Description: "Webhooks", Link: "/dashboard/settings/webhooks", Icon: "webhooks"},
		{Type: "page", Title: "Marketplace", Description: "Marketplace", Link: "/dashboard/settings/marketplace", Icon: "marketplace"},
		{Type: "page", Title: "Data Sync", Description: "Data Sync", Link: "/dashboard/settings/data-sync", Icon: "data-sync"},
	}

	for _, p := range staticPages {
		if strings.Contains(strings.ToLower(p.Title), strings.ToLower(query)) {
			results = append(results, p)
		}
	}

	// 3. Search Records
	if len(query) > 2 {
		for _, m := range modules {
			filter := bson.M{"$or": []bson.M{}}
			stringFields := []string{}
			for _, f := range m.Fields {
				if f.Type == "text" || f.Type == "email" || f.Type == "textarea" {
					stringFields = append(stringFields, f.Name)
				}
			}

			if len(stringFields) == 0 {
				continue
			}

			orConditions := []bson.M{}
			for _, fieldName := range stringFields {
				orConditions = append(orConditions, bson.M{fieldName: primitive.Regex{Pattern: query, Options: "i"}})
			}
			filter["$or"] = orConditions

			collectionName := "module_" + m.Name
			cursor, err := s.db.DB.Collection(collectionName).Find(ctx, filter, options.Find().SetLimit(3))
			if err != nil {
				continue
			}
			defer cursor.Close(ctx)

			var records []map[string]interface{}
			if err = cursor.All(ctx, &records); err == nil {
				for _, r := range records {
					title := "Unknown Record"
					if t, ok := r["name"].(string); ok {
						title = t
					} else if t, ok := r["title"].(string); ok {
						title = t
					} else if t, ok := r["subject"].(string); ok {
						title = t
					} else {
						if len(stringFields) > 0 {
							if val, ok := r[stringFields[0]].(string); ok {
								title = val
							}
						}
					}

					id := ""
					if oid, ok := r["_id"].(primitive.ObjectID); ok {
						id = oid.Hex()
					}

					results = append(results, SearchResult{
						Type:        "record",
						Title:       title,
						Description: fmt.Sprintf("%s Record", m.Label),
						Link:        fmt.Sprintf("/dashboard/modules/%s/%s", m.Name, id),
						Icon:        "file",
					})
				}
			}
		}
	}

	return results, nil
}
