package logger

import (
	"context"
	"fmt"
	"time"

	common_models "go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/database"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap/zapcore"
)

// LogEntry holds the data passed from Zap to our worker
type LogEntry struct {
	Level      zapcore.Level
	Message    string
	IpAddress  string
	CustomerId *int
	Caller     string // Function name
}

// DBLogWriter handles the async writing
type DBLogWriter struct {
	db      *mongo.Database // Use generic *gorm.DB (works for MSSQL/Oracle)
	logChan chan LogEntry
	appId   string
}

// NewDBLogWriter initializes the worker
func NewDBLogWriter(mongodb *database.MongodbDB, cfg *config.Config) *DBLogWriter {
	writer := &DBLogWriter{
		db:      mongodb.DB,                // Assuming we log to MSSQL. Swap for Oracle if needed.
		logChan: make(chan LogEntry, 1000), // Buffer 1000 logs
		appId:   cfg.AppId,                 // Fixed ApplicationId
	}

	// Start the background worker immediately
	go writer.processLogs()

	return writer
}

// AddLog is called by our Zap hook
func (w *DBLogWriter) AddLog(entry LogEntry) {
	select {
	case w.logChan <- entry:
		// Log pushed to channel
	default:
		// Channel full: drop log or print to stderr to prevent blocking the API
		fmt.Println("DB Log Channel Full! Dropping log:", entry.Message)
	}
}

func (w *DBLogWriter) processLogs() {
	for entry := range w.logChan {
		// Map Zap Level to your Int ID
		levelId := mapLevelToInt(entry.Level)

		// Handle nil CustomerId
		var customerId int
		if entry.CustomerId != nil {
			customerId = *entry.CustomerId
		}

		// Create the DB Record
		logRecord := common_models.Log{
			Message:      entry.Message,
			IpAddress:    entry.IpAddress, // Actual IP
			CustomerId:   customerId,
			LogLevelId:   levelId,
			CreatedOnUtc: time.Now().UTC(),
		}

		// Insert into DB (safely ignore errors to keep app running)
		// Note: efficient apps might batch insert here every 10 logs
		w.db.Collection("logs").InsertOne(context.Background(), logRecord)
	}
}

func mapLevelToInt(l zapcore.Level) int {
	switch l {
	case zapcore.DebugLevel:
		return 10
	case zapcore.InfoLevel:
		return 20
	case zapcore.WarnLevel:
		return 30
	case zapcore.ErrorLevel:
		return 40
	case zapcore.FatalLevel:
		return 50
	default:
		return 20
	}
}
