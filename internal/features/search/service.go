package search

import (
	"context"
	"fmt"
	"strings"

	"go-crm/internal/common/models"
	"go-crm/internal/core/role"
	"go-crm/internal/database"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"go-crm/internal/features/resource"

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
	db              *database.MongodbDB
	moduleService   module.ModuleService
	recordService   record.RecordService
	resourceService resource.ResourceService
	roleService     role.RoleService
}

func NewSearchService(
	db *database.MongodbDB,
	moduleService module.ModuleService,
	recordService record.RecordService,
	resourceService resource.ResourceService,
	roleService role.RoleService,
) SearchService {
	return &SearchServiceImpl{
		db:              db,
		moduleService:   moduleService,
		recordService:   recordService,
		resourceService: resourceService,
		roleService:     roleService,
	}
}

func (s *SearchServiceImpl) GlobalSearch(ctx context.Context, query string, userID primitive.ObjectID) ([]SearchResult, error) {
	var results []SearchResult
	query = strings.TrimSpace(query)
	if query == "" {
		return results, nil
	}

	// 1. Fetch TenantID from Context
	tenantID, _ := ctx.Value(models.TenantIDKey).(string)
	tenantOID, _ := primitive.ObjectIDFromHex(tenantID)

	// 2. Fetch permitted resources
	allResources, err := s.resourceService.ListResources(ctx)
	if err != nil {
		return results, err
	}

	var permittedModules []models.Resource
	queryLower := strings.ToLower(query)

	for _, res := range allResources {
		// Permission check
		allowed, err := s.roleService.CheckPermission(ctx, userID, res.ResourceID, "read")
		if err != nil || !allowed {
			continue
		}

		// Match query in label
		if strings.Contains(strings.ToLower(res.Label), queryLower) {
			resType := "page"
			if res.Type == "module" {
				resType = "module"
				permittedModules = append(permittedModules, res)
			} else if res.Type == "setting" || res.UI.Location == "settings" {
				resType = "page"
			}

			results = append(results, SearchResult{
				Type:        resType,
				Title:       res.Label,
				Name:        res.Key,
				Description: fmt.Sprintf("Go to %s", res.Label),
				Link:        res.Route,
				Icon:        res.Icon,
			})
		} else if res.Type == "module" {
			// Even if label doesn't match, we still want to search records if it's a module
			permittedModules = append(permittedModules, res)
		}
	}

	// 3. Search Records
	if len(query) > 2 {
		for _, m := range permittedModules {
			// Fetch full module entity to get fields for record search
			moduleEntity, err := s.moduleService.GetModuleByName(ctx, m.Key, userID)
			if err != nil {
				continue
			}

			stringFields := []string{}
			for _, f := range moduleEntity.Fields {
				if f.Type == models.FieldTypeText || f.Type == models.FieldTypeEmail || f.Type == models.FieldTypeTextArea {
					stringFields = append(stringFields, f.Name)
				}
			}

			if len(stringFields) == 0 {
				continue
			}

			orConditions := []bson.M{}
			for _, fieldName := range stringFields {
				orConditions = append(orConditions, bson.M{"data." + fieldName: primitive.Regex{Pattern: query, Options: "i"}})
			}

			// Add filters for unified collection
			filter := bson.M{
				"entity":    moduleEntity.Name,
				"__deleted": bson.M{"$ne": true},
				"$or":       orConditions,
			}
			if !tenantOID.IsZero() {
				filter["tenant_id"] = tenantOID
			}

			collectionName := "entity_records"
			cursor, err := s.db.DB.Collection(collectionName).Find(ctx, filter, options.Find().SetLimit(3))
			if err != nil {
				continue
			}
			defer cursor.Close(ctx)

			var records []map[string]any
			if err = cursor.All(ctx, &records); err == nil {
				for _, r := range records {
					data, _ := r["data"].(map[string]any)
					if data == nil {
						continue
					}

					title := "Unknown Record"
					if t, ok := data["name"].(string); ok {
						title = t
					} else if t, ok := data["title"].(string); ok {
						title = t
					} else if t, ok := data["subject"].(string); ok {
						title = t
					} else {
						if len(stringFields) > 0 {
							if val, ok := data[stringFields[0]].(string); ok {
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
						Description: fmt.Sprintf("%s Record", moduleEntity.Label),
						Link:        fmt.Sprintf("/dashboard/modules/%s/%s", moduleEntity.Name, id),
						Icon:        "file",
					})
				}
			}
		}
	}

	return results, nil
}
