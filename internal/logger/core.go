package logger

import (
	"go.uber.org/zap/zapcore"
)

// DBCore is a custom Zap Core that intercepts logs
type DBCore struct {
	zapcore.Core
	writer *DBLogWriter
}

// NewDBCore wraps an existing core (like console logger) and adds DB logging
func NewDBCore(baseCore zapcore.Core, writer *DBLogWriter) zapcore.Core {
	return &DBCore{
		Core:   baseCore,
		writer: writer,
	}
}

// Write is called for every log entry
func (c *DBCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// 1. Extract Contextual Fields (IP, CustomerId)
	var ip string
	var customerId *int

	// Loop through fields to find custom ones we attached
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
		if f.Key == "ip" {
			ip = f.String
		}
		if f.Key == "customerId" {
			id := int(f.Integer)
			customerId = &id
		}
	}

	// 2. Extract Function Name
	// entry.Caller is usually file path; Function name needs runtime lookup or Caller.Function if configured
	functionName := entry.Caller.Function
	// Note: To get "Function", Zap must be configured with AddCaller()

	// 3. Send to Async Writer
	c.writer.AddLog(LogEntry{
		Level:      entry.Level,
		Message:    entry.Message,
		IpAddress:  ip,
		CustomerId: customerId,
		Caller:     functionName,
	})

	// 4. Call the underlying core (so it still prints to Console/File)
	return c.Core.Write(entry, fields)
}

// Check decides if we should log this level
func (c *DBCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}
