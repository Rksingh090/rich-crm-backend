package cron_feature

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/automation"
	"go-crm/internal/features/email"
	"go-crm/internal/features/record"
	sync_feature "go-crm/internal/features/sync"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type CronService interface {
	CreateCronJob(ctx context.Context, cronJob *CronJob) error
	GetCronJob(ctx context.Context, id string) (*CronJob, error)
	ListCronJobs(ctx context.Context, filter map[string]interface{}) ([]CronJob, error)
	UpdateCronJob(ctx context.Context, cronJob *CronJob) error
	DeleteCronJob(ctx context.Context, id string) error
	ExecuteCronJob(ctx context.Context, id string) error
	GetCronJobLogs(ctx context.Context, cronJobID string, limit int) ([]CronJobLog, error)
	InitializeScheduler(ctx context.Context) error
	StopScheduler() error
	RegisterJob(cronJob *CronJob) error
	UnregisterJob(id string) error
}

type CronServiceImpl struct {
	repo           CronRepository
	recordRepo     record.RecordRepository
	actionExecutor automation.ActionExecutor
	auditService   audit.AuditService
	syncService    sync_feature.SyncService
	emailService   email.EmailService

	scheduler  *cron.Cron
	jobEntries map[string]cron.EntryID
	mu         sync.RWMutex
}

func NewCronService(
	repo CronRepository,
	recordRepo record.RecordRepository,
	actionExecutor automation.ActionExecutor,
	auditService audit.AuditService,
	syncService sync_feature.SyncService,
	emailService email.EmailService,
) CronService {
	return &CronServiceImpl{
		repo:           repo,
		recordRepo:     recordRepo,
		actionExecutor: actionExecutor,
		auditService:   auditService,
		syncService:    syncService,
		emailService:   emailService,
		jobEntries:     make(map[string]cron.EntryID),
	}
}

func (s *CronServiceImpl) CreateCronJob(ctx context.Context, cronJob *CronJob) error {
	if _, err := cron.ParseStandard(cronJob.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	now := time.Now()
	cronJob.CreatedAt = now
	cronJob.UpdatedAt = now

	schedule, _ := cron.ParseStandard(cronJob.Schedule)
	nextRun := schedule.Next(now)
	cronJob.NextRun = &nextRun

	if err := s.repo.Create(ctx, cronJob); err != nil {
		return err
	}

	s.auditService.LogChange(ctx, common_models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]common_models.Change{
		"cron_job": {New: cronJob},
	})

	if cronJob.Active && s.scheduler != nil {
		if err := s.RegisterJob(cronJob); err != nil {
			log.Printf("Failed to register cron job %s: %v", cronJob.ID.Hex(), err)
		}
	}

	return nil
}

func (s *CronServiceImpl) GetCronJob(ctx context.Context, id string) (*CronJob, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CronServiceImpl) ListCronJobs(ctx context.Context, filter map[string]interface{}) ([]CronJob, error) {
	return s.repo.List(ctx, filter)
}

func (s *CronServiceImpl) UpdateCronJob(ctx context.Context, cronJob *CronJob) error {
	if _, err := cron.ParseStandard(cronJob.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	schedule, _ := cron.ParseStandard(cronJob.Schedule)
	nextRun := schedule.Next(time.Now())
	cronJob.NextRun = &nextRun

	oldJob, _ := s.GetCronJob(ctx, cronJob.ID.Hex())

	if err := s.repo.Update(ctx, cronJob); err != nil {
		return err
	}

	s.auditService.LogChange(ctx, common_models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]common_models.Change{
		"cron_job": {Old: oldJob, New: cronJob},
	})

	s.UnregisterJob(cronJob.ID.Hex())

	if cronJob.Active && s.scheduler != nil {
		if err := s.RegisterJob(cronJob); err != nil {
			log.Printf("Failed to register updated cron job %s: %v", cronJob.ID.Hex(), err)
		}
	}

	return nil
}

func (s *CronServiceImpl) DeleteCronJob(ctx context.Context, id string) error {
	oldJob, _ := s.GetCronJob(ctx, id)
	s.UnregisterJob(id)
	err := s.repo.Delete(ctx, id)
	if err == nil {
		s.auditService.LogChange(ctx, common_models.AuditActionCron, "cron", id, map[string]common_models.Change{
			"cron_job": {Old: oldJob, New: "DELETED"},
		})
	}
	return err
}

func (s *CronServiceImpl) ExecuteCronJob(ctx context.Context, id string) error {
	cronJob, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cronJob == nil {
		return fmt.Errorf("cron job not found")
	}

	return s.executeCronJobInternal(ctx, cronJob)
}

func (s *CronServiceImpl) executeCronJobInternal(ctx context.Context, cronJob *CronJob) error {
	startTime := time.Now()

	logEntry := &CronJobLog{
		CronJobID:   cronJob.ID,
		CronJobName: cronJob.Name,
		StartTime:   startTime,
		Status:      "running",
	}

	if err := s.repo.CreateLog(ctx, logEntry); err != nil {
		log.Printf("Failed to create log entry for cron job %s: %v", cronJob.ID.Hex(), err)
	}

	s.auditService.LogChange(ctx, common_models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]common_models.Change{
		"status":   {New: "started"},
		"job_name": {New: cronJob.Name},
	})

	var execError error
	recordsProcessed := 0
	recordsAffected := 0

	if cronJob.ModuleID != "" {
		recordsProcessed, recordsAffected, execError = s.executeRecordBasedJob(ctx, cronJob)
	} else {
		recordsAffected, execError = s.executeNonRecordBasedJob(ctx, cronJob)
	}

	endTime := time.Now()
	logEntry.EndTime = &endTime
	logEntry.RecordsProcessed = recordsProcessed
	logEntry.RecordsAffected = recordsAffected

	if execError != nil {
		logEntry.Status = "failed"
		logEntry.Error = execError.Error()
	} else {
		logEntry.Status = "success"
	}

	if err := s.repo.UpdateLog(ctx, logEntry); err != nil {
		log.Printf("Failed to update log entry for cron job %s: %v", cronJob.ID.Hex(), err)
	}

	auditStatus := "success"
	if execError != nil {
		auditStatus = "failed"
	}
	s.auditService.LogChange(ctx, common_models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]common_models.Change{
		"status":   {New: auditStatus},
		"affected": {New: recordsAffected},
		"error":    {New: logEntry.Error},
	})

	schedule, _ := cron.ParseStandard(cronJob.Schedule)
	nextRun := schedule.Next(time.Now())
	if err := s.repo.UpdateLastRun(ctx, cronJob.ID.Hex(), startTime, &nextRun); err != nil {
		log.Printf("Failed to update last run for cron job %s: %v", cronJob.ID.Hex(), err)
	}

	return execError
}

func (s *CronServiceImpl) executeRecordBasedJob(ctx context.Context, cronJob *CronJob) (int, int, error) {
	filter := make(map[string]interface{})

	if len(cronJob.Conditions) > 0 {
		for _, condition := range cronJob.Conditions {
			switch condition.Operator {
			case OperatorEquals:
				filter[condition.Field] = condition.Value
			case OperatorNotEquals:
				filter[condition.Field] = map[string]interface{}{"$ne": condition.Value}
			case OperatorGreaterThan:
				filter[condition.Field] = map[string]interface{}{"$gt": condition.Value}
			case OperatorLessThan:
				filter[condition.Field] = map[string]interface{}{"$lt": condition.Value}
			case OperatorContains:
				filter[condition.Field] = map[string]interface{}{"$regex": condition.Value, "$options": "i"}
			}
		}
	}

	records, err := s.recordRepo.List(ctx, cronJob.ModuleID, filter, 1000, 0, "created_at", -1)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch records: %w", err)
	}

	recordsProcessed := len(records)
	recordsAffected := 0

	for _, record := range records {
		// Convert features/cron/RuleAction to features/automation/RuleAction
		autoActions := make([]automation.RuleAction, len(cronJob.Actions))
		for i, a := range cronJob.Actions {
			autoActions[i] = automation.RuleAction{
				Type:   automation.ActionType(a.Type),
				Config: a.Config,
			}
		}

		if err := s.actionExecutor.ExecuteActions(ctx, autoActions, cronJob.ModuleID, record); err != nil {
			log.Printf("Failed to execute actions for record %v: %v", record["_id"], err)
		} else {
			recordsAffected++
		}
	}

	return recordsProcessed, recordsAffected, nil
}

func (s *CronServiceImpl) executeNonRecordBasedJob(ctx context.Context, cronJob *CronJob) (int, error) {
	recordsAffected := 0

	for _, action := range cronJob.Actions {
		switch action.Type {
		case ActionSendEmail:
			log.Printf("Sending email with config: %v", action.Config)
			recordsAffected++
		case ActionWebhook:
			log.Printf("Triggering webhook with config: %v", action.Config)
			recordsAffected++
		case ActionDataSync:
			syncSettingID, ok := action.Config["sync_setting_id"].(string)
			if !ok {
				return recordsAffected, fmt.Errorf("sync_setting_id missing in action config")
			}
			if err := s.syncService.RunSync(ctx, syncSettingID); err != nil {
				return recordsAffected, err
			}
			recordsAffected++
		default:
			log.Printf("Action type %s not supported for non-record based jobs", action.Type)
		}
	}

	return recordsAffected, nil
}

func (s *CronServiceImpl) GetCronJobLogs(ctx context.Context, cronJobID string, limit int) ([]CronJobLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetLogs(ctx, cronJobID, limit)
}

func (s *CronServiceImpl) InitializeScheduler(ctx context.Context) error {
	log.Println("Initializing cron scheduler...")
	s.scheduler = cron.New()
	cronJobs, err := s.repo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active cron jobs: %w", err)
	}

	for i := range cronJobs {
		if err := s.RegisterJob(&cronJobs[i]); err != nil {
			log.Printf("Failed to register cron job %s: %v", cronJobs[i].ID.Hex(), err)
		}
	}

	s.scheduler.Start()
	return nil
}

func (s *CronServiceImpl) StopScheduler() error {
	if s.scheduler != nil {
		ctx := s.scheduler.Stop()
		<-ctx.Done()
	}
	return nil
}

func (s *CronServiceImpl) RegisterJob(cronJob *CronJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}

	cronJobID := cronJob.ID.Hex()
	jobFunc := func() {
		ctx := context.Background()
		latestCronJob, err := s.repo.GetByID(ctx, cronJobID)
		if err != nil || latestCronJob == nil || !latestCronJob.Active {
			return
		}
		s.executeCronJobInternal(ctx, latestCronJob)
	}

	entryID, err := s.scheduler.AddFunc(cronJob.Schedule, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job to scheduler: %w", err)
	}

	s.jobEntries[cronJobID] = entryID
	return nil
}

func (s *CronServiceImpl) UnregisterJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobEntries[id]; exists {
		s.scheduler.Remove(entryID)
		delete(s.jobEntries, id)
	}
	return nil
}
