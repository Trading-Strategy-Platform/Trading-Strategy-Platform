// services/historical-data-service/cmd/tools/fix_data_flags.go

package main

import (
	"context"
	"fmt"
	"log"

	"services/historical-data-service/internal/config"
	"services/historical-data-service/internal/repository"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("../../config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up logger
	logger, err := createLogger(cfg.Logging.Level)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Connect to database
	db, err := connectToDB(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repositories
	symbolRepo := repository.NewSymbolRepository(db, logger)
	downloadJobRepo := repository.NewDownloadJobRepository(db, logger)

	// Run the fix
	err = fixDataAvailabilityFlags(symbolRepo, downloadJobRepo, logger)
	if err != nil {
		logger.Fatal("Failed to fix data availability flags", zap.Error(err))
	}

	logger.Info("Successfully fixed data availability flags")
}

func fixDataAvailabilityFlags(
	symbolRepo *repository.SymbolRepository,
	downloadJobRepo *repository.DownloadJobRepository,
	logger *zap.Logger,
) error {
	ctx := context.Background()

	// Get all symbols
	symbols, err := symbolRepo.GetAllSymbols(ctx)
	if err != nil {
		return err
	}

	logger.Info("Found symbols", zap.Int("count", len(symbols)))

	// Check each symbol for data
	for _, symbol := range symbols {
		logger.Info("Checking symbol",
			zap.String("symbol", symbol.Symbol),
			zap.Int("id", symbol.ID),
			zap.Bool("dataAvailable", symbol.DataAvailable))

		// Get candle count
		count, err := downloadJobRepo.GetCandleCount(ctx, symbol.ID)
		if err != nil {
			logger.Error("Failed to get candle count",
				zap.Error(err),
				zap.Int("symbolID", symbol.ID))
			continue
		}

		// Symbol has data but flag is not set
		if count > 0 && !symbol.DataAvailable {
			logger.Info("Updating data availability flag",
				zap.String("symbol", symbol.Symbol),
				zap.Int("id", symbol.ID),
				zap.Int("candleCount", count))

			success, err := symbolRepo.UpdateDataAvailability(ctx, symbol.ID, true)
			if err != nil || !success {
				logger.Error("Failed to update data availability flag",
					zap.Error(err),
					zap.Int("symbolID", symbol.ID))
				continue
			}

			logger.Info("Updated data availability flag",
				zap.String("symbol", symbol.Symbol),
				zap.Int("id", symbol.ID))
		}

		// Symbol has no data but flag is set
		if count == 0 && symbol.DataAvailable {
			logger.Info("Clearing incorrect data availability flag",
				zap.String("symbol", symbol.Symbol),
				zap.Int("id", symbol.ID))

			success, err := symbolRepo.UpdateDataAvailability(ctx, symbol.ID, false)
			if err != nil || !success {
				logger.Error("Failed to clear data availability flag",
					zap.Error(err),
					zap.Int("symbolID", symbol.ID))
				continue
			}

			logger.Info("Cleared data availability flag",
				zap.String("symbol", symbol.Symbol),
				zap.Int("id", symbol.ID))
		}
	}

	return nil
}

func createLogger(level string) (*zap.Logger, error) {
	// Parse log level
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Create logger config
	config := zap.Config{
		Level:            zapLevel,
		Development:      false,
		Encoding:         "console", // Use console encoding for human-readable output
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

func connectToDB(dbConfig config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName,
		dbConfig.SSLMode,
	)

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

	return db, nil
}
