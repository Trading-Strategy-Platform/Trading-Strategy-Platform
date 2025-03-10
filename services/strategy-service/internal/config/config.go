// services/strategy-service/internal/config/config.go
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	sharedConfig "github.com/yourorg/trading-platform/shared/go/config"
)

// Config holds all configuration for the service
type Config struct {
	Server            ServerConfig
	Database          DatabaseConfig
	UserService       ServiceConfig
	HistoricalService ServiceConfig
	Kafka             KafkaConfig
	Logging           LoggingConfig
	ServiceKey        string
	Auth              struct {
		JWTSecret string
	}
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
	// Use shared config loader
	v, err := sharedConfig.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Add service specific defaults
	setServiceDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setServiceDefaults sets service-specific default values
func setServiceDefaults(v *viper.Viper) {
	// Strategy Service defaults
	v.SetDefault("userService.timeout", "5s")
	v.SetDefault("userService.serviceKey", "strategy-service-key")

	v.SetDefault("historicalService.timeout", "30s")
	v.SetDefault("historicalService.serviceKey", "strategy-service-key")

	v.SetDefault("kafka.topics.strategyEvents", "strategy-events")
	v.SetDefault("kafka.topics.marketplaceEvents", "marketplace-events")
}
