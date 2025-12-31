package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-crm/internal/models"
	"go-crm/internal/repository"
	"log"
	"sync"
	"time"

	"strings"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CronService interface {
	// CRUD operations
	CreateCronJob(ctx context.Context, cronJob *models.CronJob) error
	GetCronJob(ctx context.Context, id string) (*models.CronJob, error)
	ListCronJobs(ctx context.Context, filter map[string]interface{}) ([]models.CronJob, error)
	UpdateCronJob(ctx context.Context, cronJob *models.CronJob) error
	DeleteCronJob(ctx context.Context, id string) error

	// Execution
	ExecuteCronJob(ctx context.Context, id string) error

	// Logs
	GetCronJobLogs(ctx context.Context, cronJobID string, limit int) ([]models.CronJobLog, error)

	// Scheduler management
	InitializeScheduler(ctx context.Context) error
	StopScheduler() error
	RegisterJob(cronJob *models.CronJob) error
	UnregisterJob(id string) error
}

type CronServiceImpl struct {
	repo           repository.CronRepository
	recordRepo     repository.RecordRepository
	actionExecutor ActionExecutor
	auditService   AuditService
	syncService    SyncService
	reportService  ReportService
	emailService   EmailService

	scheduler  *cron.Cron
	jobEntries map[string]cron.EntryID // Maps cronJob ID to cron entry ID
	mu         sync.RWMutex
}

func NewCronService(
	repo repository.CronRepository,
	recordRepo repository.RecordRepository,
	actionExecutor ActionExecutor,
	auditService AuditService,
	syncService SyncService,
	reportService ReportService,
	emailService EmailService,
) CronService {
	return &CronServiceImpl{
		repo:           repo,
		recordRepo:     recordRepo,
		actionExecutor: actionExecutor,
		auditService:   auditService,
		syncService:    syncService,
		reportService:  reportService,
		emailService:   emailService,
		jobEntries:     make(map[string]cron.EntryID),
	}
}

func (s *CronServiceImpl) CreateCronJob(ctx context.Context, cronJob *models.CronJob) error {
	// Validate cron expression
	if _, err := cron.ParseStandard(cronJob.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Set timestamps
	now := time.Now()
	cronJob.CreatedAt = now
	cronJob.UpdatedAt = now

	// Calculate next run time
	schedule, _ := cron.ParseStandard(cronJob.Schedule)
	nextRun := schedule.Next(now)
	cronJob.NextRun = &nextRun

	// Create in database
	if err := s.repo.Create(ctx, cronJob); err != nil {
		return err
	}

	// Audit Log
	s.auditService.LogChange(ctx, models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]models.Change{
		"cron_job": {New: cronJob},
	})

	// Register with scheduler if active
	if cronJob.Active && s.scheduler != nil {
		if err := s.RegisterJob(cronJob); err != nil {
			log.Printf("Failed to register cron job %s: %v", cronJob.ID.Hex(), err)
		}
	}

	return nil
}

func (s *CronServiceImpl) GetCronJob(ctx context.Context, id string) (*models.CronJob, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CronServiceImpl) ListCronJobs(ctx context.Context, filter map[string]interface{}) ([]models.CronJob, error) {
	return s.repo.List(ctx, filter)
}

func (s *CronServiceImpl) UpdateCronJob(ctx context.Context, cronJob *models.CronJob) error {
	// Validate cron expression
	if _, err := cron.ParseStandard(cronJob.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Update next run time
	schedule, _ := cron.ParseStandard(cronJob.Schedule)
	nextRun := schedule.Next(time.Now())
	cronJob.NextRun = &nextRun

	// Get old job for audit
	oldJob, _ := s.GetCronJob(ctx, cronJob.ID.Hex())

	// Update in database
	if err := s.repo.Update(ctx, cronJob); err != nil {
		return err
	}

	// Audit Log
	s.auditService.LogChange(ctx, models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]models.Change{
		"cron_job": {Old: oldJob, New: cronJob},
	})

	// Unregister old job
	s.UnregisterJob(cronJob.ID.Hex())

	// Register new job if active
	if cronJob.Active && s.scheduler != nil {
		if err := s.RegisterJob(cronJob); err != nil {
			log.Printf("Failed to register updated cron job %s: %v", cronJob.ID.Hex(), err)
		}
	}

	return nil
}

func (s *CronServiceImpl) DeleteCronJob(ctx context.Context, id string) error {
	// Get old job for audit
	oldJob, _ := s.GetCronJob(ctx, id)

	// Unregister from scheduler
	s.UnregisterJob(id)

	// Delete from database
	err := s.repo.Delete(ctx, id)
	if err == nil {
		s.auditService.LogChange(ctx, models.AuditActionCron, "cron", id, map[string]models.Change{
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

func (s *CronServiceImpl) executeCronJobInternal(ctx context.Context, cronJob *models.CronJob) error {
	startTime := time.Now()

	// Create log entry
	logEntry := &models.CronJobLog{
		CronJobID:   cronJob.ID,
		CronJobName: cronJob.Name,
		StartTime:   startTime,
		Status:      "running",
	}

	if err := s.repo.CreateLog(ctx, logEntry); err != nil {
		log.Printf("Failed to create log entry for cron job %s: %v", cronJob.ID.Hex(), err)
	}

	// Audit Log Start
	s.auditService.LogChange(ctx, models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]models.Change{
		"status":   {New: "started"},
		"job_name": {New: cronJob.Name},
	})

	var execError error
	recordsProcessed := 0
	recordsAffected := 0

	// Execute the job
	if cronJob.ModuleID != "" {
		// Record-based execution
		recordsProcessed, recordsAffected, execError = s.executeRecordBasedJob(ctx, cronJob)
	} else {
		// Non-record based execution (e.g., send email, webhook)
		recordsAffected, execError = s.executeNonRecordBasedJob(ctx, cronJob)
	}

	// Update log entry
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

	// Audit Log End
	auditStatus := "success"
	if execError != nil {
		auditStatus = "failed"
	}
	s.auditService.LogChange(ctx, models.AuditActionCron, "cron", cronJob.ID.Hex(), map[string]models.Change{
		"status":   {New: auditStatus},
		"affected": {New: recordsAffected},
		"error":    {New: logEntry.Error},
	})

	// Update last run and next run times
	schedule, _ := cron.ParseStandard(cronJob.Schedule)
	nextRun := schedule.Next(time.Now())
	if err := s.repo.UpdateLastRun(ctx, cronJob.ID.Hex(), startTime, &nextRun); err != nil {
		log.Printf("Failed to update last run for cron job %s: %v", cronJob.ID.Hex(), err)
	}

	return execError
}

func (s *CronServiceImpl) executeRecordBasedJob(ctx context.Context, cronJob *models.CronJob) (int, int, error) {
	// Query records from the module
	filter := make(map[string]interface{})

	// Apply conditions to filter
	if len(cronJob.Conditions) > 0 {
		for _, condition := range cronJob.Conditions {
			// Build filter based on conditions
			// This is a simplified version - you may need more sophisticated filtering
			switch condition.Operator {
			case models.OperatorEquals:
				filter[condition.Field] = condition.Value
			case models.OperatorNotEquals:
				filter[condition.Field] = map[string]interface{}{"$ne": condition.Value}
			case models.OperatorGreaterThan:
				filter[condition.Field] = map[string]interface{}{"$gt": condition.Value}
			case models.OperatorLessThan:
				filter[condition.Field] = map[string]interface{}{"$lt": condition.Value}
			case models.OperatorContains:
				filter[condition.Field] = map[string]interface{}{"$regex": condition.Value, "$options": "i"}
			}
		}
	}

	// Fetch records
	records, err := s.recordRepo.List(ctx, cronJob.ModuleID, filter, 1000, 0, "created_at", -1) // Limit to 1000 records
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch records: %w", err)
	}

	recordsProcessed := len(records)
	recordsAffected := 0

	// Execute actions for each record
	for _, record := range records {
		if err := s.executeActionsForRecord(ctx, cronJob, record); err != nil {
			log.Printf("Failed to execute actions for record %v: %v", record["_id"], err)
		} else {
			recordsAffected++
		}
	}

	return recordsProcessed, recordsAffected, nil
}

func (s *CronServiceImpl) executeNonRecordBasedJob(ctx context.Context, cronJob *models.CronJob) (int, error) {
	// Execute actions without a specific record context
	recordsAffected := 0

	for _, action := range cronJob.Actions {
		switch action.Type {
		case models.ActionSendEmail:
			// Send email using config
			if err := s.sendEmailAction(ctx, action.Config); err != nil {
				return recordsAffected, err
			}
			recordsAffected++

		case models.ActionWebhook:
			// Trigger webhook
			if err := s.triggerWebhookAction(ctx, action.Config); err != nil {
				return recordsAffected, err
			}
			recordsAffected++

		case models.ActionDataSync:
			// Trigger data sync
			syncSettingID, ok := action.Config["sync_setting_id"].(string)
			if !ok {
				return recordsAffected, fmt.Errorf("sync_setting_id missing in action config")
			}
			if err := s.syncService.RunSync(ctx, syncSettingID); err != nil {
				return recordsAffected, err
			}
			recordsAffected++

		case models.ActionSendReport:
			// Send automated report using local implementation to avoid cycle
			if err := s.executeSendReport(ctx, action.Config); err != nil {
				return recordsAffected, err
			}
			recordsAffected++

		default:
			log.Printf("Action type %s not supported for non-record based jobs", action.Type)
		}
	}

	return recordsAffected, nil
}

func (s *CronServiceImpl) executeActionsForRecord(ctx context.Context, cronJob *models.CronJob, record map[string]interface{}) error {
	// Use centralized ActionExecutor
	return s.actionExecutor.ExecuteActions(ctx, cronJob.Actions, cronJob.ModuleID, record)
}

func (s *CronServiceImpl) sendEmailAction(_ context.Context, config map[string]interface{}) error {
	// Implement email sending logic
	// This would use the EmailService
	log.Printf("Sending email with config: %v", config)
	return nil
}

func (s *CronServiceImpl) triggerWebhookAction(_ context.Context, config map[string]interface{}) error {
	// Implement webhook trigger logic
	log.Printf("Triggering webhook with config: %v", config)
	return nil
}

func (s *CronServiceImpl) GetCronJobLogs(ctx context.Context, cronJobID string, limit int) ([]models.CronJobLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetLogs(ctx, cronJobID, limit)
}

func (s *CronServiceImpl) InitializeScheduler(ctx context.Context) error {
	log.Println("Initializing cron scheduler...")

	// Create scheduler with standard 5-field cron expressions (minute hour day month weekday)
	s.scheduler = cron.New()

	// Load all active cron jobs
	cronJobs, err := s.repo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active cron jobs: %w", err)
	}

	log.Printf("Found %d active cron jobs to schedule", len(cronJobs))

	// Register each job
	for i := range cronJobs {
		cronJob := &cronJobs[i]
		if err := s.RegisterJob(cronJob); err != nil {
			log.Printf("Failed to register cron job %s: %v", cronJob.ID.Hex(), err)
		}
	}

	// Start the scheduler
	s.scheduler.Start()
	log.Println("Cron scheduler started successfully")

	return nil
}

func (s *CronServiceImpl) StopScheduler() error {
	if s.scheduler != nil {
		log.Println("Stopping cron scheduler...")
		ctx := s.scheduler.Stop()
		<-ctx.Done()
		log.Println("Cron scheduler stopped")
	}
	return nil
}

func (s *CronServiceImpl) RegisterJob(cronJob *models.CronJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}

	// Create a closure that captures the cronJob ID
	cronJobID := cronJob.ID.Hex()
	jobFunc := func() {
		ctx := context.Background()

		// Fetch the latest version of the cron job
		latestCronJob, err := s.repo.GetByID(ctx, cronJobID)
		if err != nil {
			log.Printf("Failed to fetch cron job %s: %v", cronJobID, err)
			return
		}

		if latestCronJob == nil || !latestCronJob.Active {
			log.Printf("Cron job %s is no longer active, skipping execution", cronJobID)
			return
		}

		log.Printf("Executing cron job: %s (%s)", latestCronJob.Name, cronJobID)

		if err := s.executeCronJobInternal(ctx, latestCronJob); err != nil {
			log.Printf("Cron job %s execution failed: %v", cronJobID, err)
		} else {
			log.Printf("Cron job %s executed successfully", cronJobID)
		}
	}

	// Add job to scheduler
	entryID, err := s.scheduler.AddFunc(cronJob.Schedule, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job to scheduler: %w", err)
	}

	s.jobEntries[cronJobID] = entryID
	log.Printf("Registered cron job: %s (%s) with schedule: %s", cronJob.Name, cronJobID, cronJob.Schedule)

	return nil
}

func (s *CronServiceImpl) UnregisterJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scheduler == nil {
		return nil
	}

	if entryID, exists := s.jobEntries[id]; exists {
		s.scheduler.Remove(entryID)
		delete(s.jobEntries, id)
		log.Printf("Unregistered cron job: %s", id)
	}

	return nil
}

// Helper to format cron job output
func formatCronJobOutput(data interface{}) string {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(bytes)
}

// executeSendReport generates a report and emails it as an attachment
func (s *CronServiceImpl) executeSendReport(ctx context.Context, config map[string]interface{}) error {
	reportID, _ := config["report_id"].(string)
	format, _ := config["format"].(string)
	recipientsStr, _ := config["recipients"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)

	if reportID == "" {
		return fmt.Errorf("report_id is required for send_report action")
	}

	if format == "" {
		format = "csv"
	}

	var recipients []string
	if recipientsStr != "" {
		// Expecting comma-separated emails
		for _, email := range strings.Split(recipientsStr, ",") {
			email = strings.TrimSpace(email)
			if email != "" {
				recipients = append(recipients, email)
			}
		}
	}

	if len(recipients) == 0 {
		return fmt.Errorf("at least one recipient is required for send_report action")
	}

	if subject == "" {
		subject = "Scheduled Report"
	}
	if body == "" {
		body = "Please find the attached report."
	}

	// 1. Export Report
	userID := primitive.NilObjectID
	if creatorIDStr, ok := config["created_by"].(string); ok {
		if oid, err := primitive.ObjectIDFromHex(creatorIDStr); err == nil {
			userID = oid
		}
	}

	data, filename, err := s.reportService.ExportReport(ctx, reportID, format, userID)
	if err != nil {
		return fmt.Errorf("failed to export report: %w", err)
	}

	// 2. Send Email
	if err := s.emailService.SendEmailWithAttachment(ctx, recipients, subject, body, filename, data); err != nil {
		return fmt.Errorf("failed to send report email: %w", err)
	}

	log.Printf("Successfully sent report %s to %v", reportID, recipients)
	return nil
}
