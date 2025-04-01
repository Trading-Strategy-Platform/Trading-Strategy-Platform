package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server            ServerConfig
	Database          DatabaseConfig
	UserService       ServiceConfig
	HistoricalService ServiceConfig
	MediaService      ServiceConfig // Added for media service
	Kafka             KafkaConfig
	Logging           LoggingConfig
}

// ServerConfig holds server specific configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database specific configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ServiceConfig holds configuration for external services
type ServiceConfig struct {
	URL        string
	Timeout    time.Duration
	ServiceKey string
}

// KafkaConfig holds Kafka specific configuration
type KafkaConfig struct {
	Brokers string
	Topics  map[string]string
}

// LoggingConfig holds logging specific configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// LoadConfig loads the configuration from file and environment variables
func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Environment variables override
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.readTimeout", "10s")
	v.SetDefault("server.writeTimeout", "10s")
	v.SetDefault("server.idleTimeout", "120s")

	// Database defaults
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.maxOpenConns", 25)
	v.SetDefault("database.maxIdleConns", 5)
	v.SetDefault("database.connMaxLifetime", "30m")

	// User Service defaults
	v.SetDefault("userService.timeout", "5s")
	v.SetDefault("userService.serviceKey", "strategy-service-key")

	// Historical Service defaults
	v.SetDefault("historicalService.timeout", "30s")
	v.SetDefault("historicalService.serviceKey", "strategy-service-key")

	// Media Service defaults
	v.SetDefault("mediaService.url", "http://media-service:8085")
	v.SetDefault("mediaService.timeout", "30s")
	v.SetDefault("mediaService.serviceKey", "media-service-key")

	// Kafka topic defaults
	v.SetDefault("kafka.topics.strategyEvents", "strategy-events")
	v.SetDefault("kafka.topics.marketplaceEvents", "marketplace-events")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}
