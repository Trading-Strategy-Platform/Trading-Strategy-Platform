package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds the configuration for the logger
type Config struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputPath string `json:"output_path"`
	AppName    string `json:"app_name"`
	AppVersion string `json:"app_version"`
}

// NewLogger creates a new logger with the provided configuration
func NewLogger(config *Config) (*zap.Logger, error) {
	// Set default level if not provided
	level := zap.InfoLevel
	switch config.Level {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	}

	// Set encoder based on format
	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if config.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Set output path
	var output zapcore.WriteSyncer
	if config.OutputPath == "stdout" || config.OutputPath == "" {
		output = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewCore(encoder, output, zap.NewAtomicLevelAt(level))

	// Add common fields
	logger := zap.New(core).With(
		zap.String("app", config.AppName),
		zap.String("version", config.AppVersion),
	)

	// Replace global logger
	zap.ReplaceGlobals(logger)

	return logger, nil
}

// Close flushes any buffered log entries
func Close(logger *zap.Logger) error {
	return logger.Sync()
}
