package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SyncService interface {
	CreateSetting(ctx context.Context, setting *SyncSetting) error
	GetSetting(ctx context.Context, id string) (*SyncSetting, error)
	ListSettings(ctx context.Context) ([]SyncSetting, error)
	UpdateSetting(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteSetting(ctx context.Context, id string) error
	RunSync(ctx context.Context, id string) error
	ListLogs(ctx context.Context, settingID string, limit int64) ([]SyncLog, error)
}

type SyncServiceImpl struct {
	SyncRepo     SyncSettingRepository
	LogRepo      SyncLogRepository
	RecordRepo   record.RecordRepository
	ModuleRepo   module.ModuleRepository
	AuditService audit.AuditService
}

func NewSyncService(syncRepo SyncSettingRepository, logRepo SyncLogRepository, recordRepo record.RecordRepository, moduleRepo module.ModuleRepository, auditService audit.AuditService) SyncService {
	return &SyncServiceImpl{
		SyncRepo:     syncRepo,
		LogRepo:      logRepo,
		RecordRepo:   recordRepo,
		ModuleRepo:   moduleRepo,
		AuditService: auditService,
	}
}

func (s *SyncServiceImpl) CreateSetting(ctx context.Context, setting *SyncSetting) error {
	err := s.SyncRepo.Create(ctx, setting)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSettings, "data_sync", setting.Name, map[string]common_models.Change{
			"sync_setting": {
				New: setting,
			},
		})
	}
	return err
}

func (s *SyncServiceImpl) GetSetting(ctx context.Context, id string) (*SyncSetting, error) {
	return s.SyncRepo.Get(ctx, id)
}

func (s *SyncServiceImpl) ListSettings(ctx context.Context) ([]SyncSetting, error) {
	return s.SyncRepo.List(ctx)
}

func (s *SyncServiceImpl) UpdateSetting(ctx context.Context, id string, updates map[string]interface{}) error {
	oldSetting, _ := s.GetSetting(ctx, id)

	err := s.SyncRepo.Update(ctx, id, updates)
	if err == nil {
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSettings, "data_sync", id, map[string]common_models.Change{
			"sync_setting": {
				Old: oldSetting,
				New: updates,
			},
		})
	}
	return err
}

func (s *SyncServiceImpl) DeleteSetting(ctx context.Context, id string) error {
	oldSetting, _ := s.GetSetting(ctx, id)

	err := s.SyncRepo.Delete(ctx, id)
	if err == nil {
		name := id
		if oldSetting != nil {
			name = oldSetting.Name
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSettings, "data_sync", name, map[string]common_models.Change{
			"sync_setting": {
				Old: oldSetting,
				New: "DELETED",
			},
		})
	}
	return err
}

func (s *SyncServiceImpl) ListLogs(ctx context.Context, settingID string, limit int64) ([]SyncLog, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.LogRepo.List(ctx, settingID, limit)
}

func (s *SyncServiceImpl) RunSync(ctx context.Context, id string) error {
	setting, err := s.SyncRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	return s.executeSync(setting)
}

func (s *SyncServiceImpl) executeSync(setting *SyncSetting) error {
	ctx := context.Background()

	log := &SyncLog{
		SyncSettingID: setting.ID,
		StartTime:     time.Now(),
		Status:        "in_progress",
	}
	_ = s.LogRepo.Create(ctx, log)

	_ = s.AuditService.LogChange(ctx, common_models.AuditActionSync, "data_sync", setting.Name, map[string]common_models.Change{
		"status": {New: "started"},
	})

	var totalProcessed int
	var syncError error

	defer func() {
		log.EndTime = time.Now()
		if syncError != nil {
			log.Status = "failed"
			log.Error = syncError.Error()
		} else {
			log.Status = "success"
		}
		log.ProcessedCount = totalProcessed

		updates := map[string]interface{}{
			"last_sync_at": time.Now(),
		}
		_ = s.SyncRepo.Update(ctx, setting.ID.Hex(), updates)
		_ = s.LogRepo.Update(ctx, log)

		auditStatus := "success"
		if syncError != nil {
			auditStatus = "failed"
		}
		_ = s.AuditService.LogChange(ctx, common_models.AuditActionSync, "data_sync", setting.Name, map[string]common_models.Change{
			"status":    {New: auditStatus},
			"processed": {New: totalProcessed},
			"error":     {New: log.Error},
		})
	}()

	for _, moduleConfig := range setting.Modules {
		processed, err := s.syncModule(ctx, setting, moduleConfig)
		if err != nil {
			syncError = err
			break
		}
		totalProcessed += processed

		if moduleConfig.SyncDeletes {
			deleted, err := s.syncDeletions(ctx, setting, moduleConfig)
			if err != nil {
				syncError = err
				break
			}
			totalProcessed += deleted
		}
	}

	return syncError
}

func (s *SyncServiceImpl) syncModule(ctx context.Context, setting *SyncSetting, config ModuleSyncConfig) (int, error) {
	filters := bson.M{
		"updated_at": bson.M{"$gt": setting.LastSyncAt},
	}

	totalSynced := 0
	page := int64(1)
	limit := int64(1000)

	for {
		offset := (page - 1) * limit
		records, err := s.RecordRepo.List(ctx, config.ModuleName, filters, limit, offset, "updated_at", 1)
		if err != nil {
			return totalSynced, fmt.Errorf("failed to fetch records for %s on page %d: %v", config.ModuleName, page, err)
		}

		if len(records) == 0 {
			break
		}

		batchCount := 0
		var batchErr error

		switch setting.TargetDBType {
		case "postgres":
			batchCount, batchErr = s.syncToPostgres(records, setting.TargetDBConfig, config)
		case "mongodb":
			batchCount, batchErr = s.syncToMongoDB(records, setting.TargetDBConfig, config)
		default:
			return totalSynced, fmt.Errorf("unsupported target DB type: %s", setting.TargetDBType)
		}

		if batchErr != nil {
			return totalSynced, batchErr
		}

		totalSynced += batchCount

		if len(records) < int(limit) {
			break
		}
		page++
	}
	return totalSynced, nil
}

func (s *SyncServiceImpl) syncDeletions(ctx context.Context, setting *SyncSetting, config ModuleSyncConfig) (int, error) {
	filters := map[string]interface{}{
		"action":    common_models.AuditActionDelete,
		"module":    config.ModuleName,
		"timestamp": map[string]interface{}{"$gt": setting.LastSyncAt},
	}

	totalDeleted := 0
	page := int64(1)
	limit := int64(1000)

	for {
		logs, err := s.AuditService.ListLogs(ctx, filters, page, limit)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to fetch delete logs for %s on page %d: %v", config.ModuleName, page, err)
		}

		if len(logs) == 0 {
			break
		}

		var deletedIDs []string
		for _, log := range logs {
			deletedIDs = append(deletedIDs, log.RecordID)
		}

		batchCount := 0
		var batchErr error

		switch setting.TargetDBType {
		case "postgres":
			batchCount, batchErr = s.syncToPostgresDelete(deletedIDs, setting.TargetDBConfig, config)
		case "mongodb":
			batchCount, batchErr = s.syncToMongoDBDelete(deletedIDs, setting.TargetDBConfig, config)
		default:
			return totalDeleted, fmt.Errorf("unsupported target DB type for deletion: %s", setting.TargetDBType)
		}

		if batchErr != nil {
			return totalDeleted, batchErr
		}

		totalDeleted += batchCount

		if len(logs) < int(limit) {
			break
		}
		page++
	}
	return totalDeleted, nil
}

func (s *SyncServiceImpl) syncToPostgres(records []map[string]any, dbConfig map[string]string, moduleConfig ModuleSyncConfig) (int, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfig["host"], dbConfig["port"], dbConfig["user"], dbConfig["password"], dbConfig["database"])

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return 0, fmt.Errorf("failed to ping postgres: %v", err)
	}

	tableName := moduleConfig.ModuleName
	count := 0

	for _, record := range records {
		columns := []string{}
		values := []interface{}{}
		updateExprs := []string{}
		placeholders := []string{}

		for crmField, targetCol := range moduleConfig.Mapping {
			val, ok := record[crmField]
			if !ok && crmField == "id" {
				val = record["_id"]
			}

			if oid, ok := val.(interface{ Hex() string }); ok {
				val = oid.Hex()
			}

			columns = append(columns, targetCol)
			values = append(values, val)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)))
			if targetCol != "id" {
				updateExprs = append(updateExprs, fmt.Sprintf("%s = $%d", targetCol, len(values)))
			}
		}

		if len(columns) == 0 {
			continue
		}

		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO UPDATE SET %s",
			tableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
			strings.Join(updateExprs, ", "),
		)

		_, err := db.Exec(query, values...)
		if err != nil {
			continue
		}
		count++
	}
	return count, nil
}

func (s *SyncServiceImpl) syncToMongoDB(records []map[string]any, dbConfig map[string]string, moduleConfig ModuleSyncConfig) (int, error) {
	uri := dbConfig["uri"]
	if uri == "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s",
			dbConfig["user"], dbConfig["password"], dbConfig["host"], dbConfig["port"], dbConfig["database"])
		if dbConfig["user"] == "" {
			uri = fmt.Sprintf("mongodb://%s:%s/%s", dbConfig["host"], dbConfig["port"], dbConfig["database"])
		}
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return 0, fmt.Errorf("failed to connect to external mongodb: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(dbConfig["database"]).Collection(moduleConfig.ModuleName)

	var modelsList []mongo.WriteModel
	for _, record := range records {
		mapped := bson.M{}
		for crmField, targetField := range moduleConfig.Mapping {
			if val, ok := record[crmField]; ok {
				mapped[targetField] = val
			} else if crmField == "id" {
				mapped[targetField] = record["_id"]
			}
		}

		if len(mapped) == 0 {
			continue
		}

		id := mapped["id"]
		if id == nil {
			id = record["_id"]
		}

		upsert := mongo.NewReplaceOneModel().
			SetFilter(bson.M{"id": id}).
			SetReplacement(mapped).
			SetUpsert(true)
		modelsList = append(modelsList, upsert)
	}

	if len(modelsList) == 0 {
		return 0, nil
	}

	res, err := collection.BulkWrite(context.Background(), modelsList)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk write to mongodb: %v", err)
	}

	return int(res.UpsertedCount + res.ModifiedCount + res.MatchedCount), nil
}

func (s *SyncServiceImpl) syncToPostgresDelete(ids []string, dbConfig map[string]string, moduleConfig ModuleSyncConfig) (int, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfig["host"], dbConfig["port"], dbConfig["user"], dbConfig["password"], dbConfig["database"])

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if len(ids) == 0 {
		return 0, nil
	}

	tableName := moduleConfig.ModuleName
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", tableName, strings.Join(placeholders, ","))
	res, err := db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete from postgres: %v", err)
	}

	count, _ := res.RowsAffected()
	return int(count), nil
}

func (s *SyncServiceImpl) syncToMongoDBDelete(ids []string, dbConfig map[string]string, moduleConfig ModuleSyncConfig) (int, error) {
	uri := dbConfig["uri"]
	if uri == "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s",
			dbConfig["user"], dbConfig["password"], dbConfig["host"], dbConfig["port"], dbConfig["database"])
		if dbConfig["user"] == "" {
			uri = fmt.Sprintf("mongodb://%s:%s/%s", dbConfig["host"], dbConfig["port"], dbConfig["database"])
		}
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return 0, fmt.Errorf("failed to connect to external mongodb: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(dbConfig["database"]).Collection(moduleConfig.ModuleName)

	if len(ids) == 0 {
		return 0, nil
	}

	filter := bson.M{"id": bson.M{"$in": ids}}
	res, err := collection.DeleteMany(context.Background(), filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete from mongodb: %v", err)
	}

	return int(res.DeletedCount), nil
}
