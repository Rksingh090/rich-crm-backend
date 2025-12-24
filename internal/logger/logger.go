package logger

import (
	"go-crm/internal/config"
	"go-crm/internal/database" // Import to get DB connection

	"go.uber.org/zap"
)

// NewLogger now requires Database to pass to the DB Writer
func NewLogger(cfg *config.Config, mssql *database.MongodbDB) (*zap.Logger, error) {

	// 1. Setup Base Config (Console/JSON)
	var zapConfig zap.Config
	if cfg.DBName == "production" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Important: Enable Caller to get Function Name
	zapConfig.EncoderConfig.FunctionKey = "func"

	// Build the base logger
	baseLogger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	// 2. Create our Async DB Writer
	dbWriter := NewDBLogWriter(mssql, cfg)

	// 3. Wrap the Core
	// We replace the logger's core with our "Tee" core (sends to both console and DB)
	finalCore := NewDBCore(baseLogger.Core(), dbWriter)

	// 4. Return new Logger with AddCaller enabled
	return zap.New(finalCore, zap.AddCaller()), nil
}
