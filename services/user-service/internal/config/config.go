package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Media    ServiceConfig
	Kafka    KafkaConfig
	Redis    RedisConfig
	Logging  LoggingConfig
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

// AuthConfig holds authentication specific configuration
type AuthConfig struct {
	JWTSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

// KafkaConfig holds Kafka specific configuration
type KafkaConfig struct {
	Brokers  []string
	Enabled  bool
	ClientID string
}

// RedisConfig holds Redis specific configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	Enabled  bool
}

// LoggingConfig holds logging specific configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// ServiceConfig holds configuration for external services
type ServiceConfig struct {
	URL        string
	Timeout    time.Duration
	ServiceKey string
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

	// Read from environment variables
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

	// Auth defaults
	v.SetDefault("auth.accessTokenDuration", "15m")
	v.SetDefault("auth.refreshTokenDuration", "7d")

	// Kafka topic defaults
	v.SetDefault("kafka.topics.notifications", "user-notifications")
	v.SetDefault("kafka.topics.events", "user-events")

	// Redis defaults
	v.SetDefault("redis.sessionPrefix", "user-session:")
	v.SetDefault("redis.sessionDuration", "24h")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}
