package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	sharedConfig "github.com/yourorg/trading-platform/shared/go/config"
)

// Config holds all configuration for the API Gateway
type Config struct {
	Server              ServerConfig
	UserService         ServiceConfig
	StrategyService     ServiceConfig
	HistoricalService   ServiceConfig
	RateLimit           RateLimitConfig
	Logging             LoggingConfig
	Auth                AuthConfig
	ExecutionService    ServiceConfig
	NotificationService ServiceConfig
	Kafka               KafkaConfig
}

// ServerConfig holds server specific configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// ServiceConfig holds configuration for downstream services
type ServiceConfig struct {
	URL     string
	Timeout time.Duration
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled            bool
	RequestsPerMinute  int
	BurstSize          int
	ClientIPHeaderName string
}

// LoggingConfig holds logging specific configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled            bool     `yaml:"enabled"`
	JWTSecret          string   `yaml:"jwt_secret"`
	ExcludedPaths      []string `yaml:"excluded_paths"`
	PublicPaths        []string `yaml:"public_paths"`
	AdminRequiredPaths []string `yaml:"admin_required_paths"`
}

// KafkaConfig holds configuration for Kafka
type KafkaConfig struct {
	Brokers string
	Topics  map[string]string
	Enabled bool
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
	// Service defaults
	v.SetDefault("userService.url", "http://user-service:8080")
	v.SetDefault("userService.timeout", "10s")

	v.SetDefault("strategyService.url", "http://strategy-service:8080")
	v.SetDefault("strategyService.timeout", "10s")

	v.SetDefault("historicalService.url", "http://historical-data-service:8080")
	v.SetDefault("historicalService.timeout", "30s")

	// Rate limit defaults
	v.SetDefault("rateLimit.enabled", false)
	v.SetDefault("rateLimit.requestsPerMinute", 60)
	v.SetDefault("rateLimit.burstSize", 10)
	v.SetDefault("rateLimit.clientIPHeaderName", "X-Real-IP")

	// Auth defaults
	v.SetDefault("auth.enabled", true)

	// Kafka defaults
	v.SetDefault("kafka.enabled", true)
	v.SetDefault("kafka.brokers", "kafka:9092")
	v.SetDefault("kafka.topics.userEvents", "user-events")
	v.SetDefault("kafka.topics.apiMetrics", "api-metrics")
	v.SetDefault("kafka.topics.systemEvents", "system-events")
}
