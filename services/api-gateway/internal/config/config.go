package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the API Gateway
type Config struct {
	Server            ServerConfig
	UserService       ServiceConfig
	StrategyService   ServiceConfig
	HistoricalService ServiceConfig
	RateLimit         RateLimitConfig
	Logging           LoggingConfig
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

	// Service defaults
	v.SetDefault("userService.timeout", "10s")
	v.SetDefault("strategyService.timeout", "10s")
	v.SetDefault("historicalService.timeout", "30s")

	// Rate limit defaults
	v.SetDefault("rateLimit.enabled", false)
	v.SetDefault("rateLimit.requestsPerMinute", 60)
	v.SetDefault("rateLimit.burstSize", 10)
	v.SetDefault("rateLimit.clientIPHeaderName", "X-Real-IP")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}
