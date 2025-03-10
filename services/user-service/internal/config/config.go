package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	sharedConfig "github.com/yourorg/trading-platform/shared/go/config"
)

// Config holds all configuration for the service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
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

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret       string `yaml:"jwt_secret"`
	AccessTokenTTL  int    `yaml:"access_token_ttl"`  // TTL in hours
	RefreshTokenTTL int    `yaml:"refresh_token_ttl"` // TTL in hours
}

// KafkaConfig holds Kafka specific configuration
type KafkaConfig struct {
	Brokers string
	Topics  map[string]string
}

// RedisConfig holds Redis specific configuration
type RedisConfig struct {
	URL             string
	SessionPrefix   string
	SessionDuration time.Duration
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
	// Only set service-specific defaults that aren't in the shared defaults
	v.SetDefault("kafka.topics.notifications", "user-notifications")
	v.SetDefault("kafka.topics.events", "user-events")
	v.SetDefault("redis.sessionPrefix", "user-session:")
	v.SetDefault("redis.sessionDuration", "24h")
}
