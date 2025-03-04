package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load loads configuration from file and environment variables
func Load(path string) (*viper.Viper, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Ignore file not found error
	}

	// Override with environment variables
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return v, nil
}

// LoadWithDefaults loads configuration with default values
func LoadWithDefaults(path string, defaults map[string]interface{}) (*viper.Viper, error) {
	v := viper.New()

	// Set provided defaults
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	// Read config file
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Ignore file not found error
	}

	// Override with environment variables
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return v, nil
}

// RequiredEnvs checks if required environment variables are set
func RequiredEnvs(envs []string) error {
	var missing []string

	for _, env := range envs {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.timeout.read", "10s")
	v.SetDefault("server.timeout.write", "10s")
	v.SetDefault("server.timeout.idle", "120s")

	// Database defaults
	v.SetDefault("database.pool.max_open", 25)
	v.SetDefault("database.pool.max_idle", 5)
	v.SetDefault("database.pool.max_lifetime", "30m")
	v.SetDefault("database.sslmode", "disable")

	// JWT defaults
	v.SetDefault("auth.jwt.access_duration", "15m")
	v.SetDefault("auth.jwt.refresh_duration", "7d")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Kafka defaults
	v.SetDefault("kafka.consumer.min_bytes", 10e3)
	v.SetDefault("kafka.consumer.max_bytes", 10e6)
	v.SetDefault("kafka.consumer.max_wait", "500ms")
	v.SetDefault("kafka.producer.timeout", "10s")
	v.SetDefault("kafka.producer.batch.size", 100)
	v.SetDefault("kafka.producer.batch.timeout", "1s")

	// Redis defaults
	v.SetDefault("redis.pool.max_idle", 10)
	v.SetDefault("redis.pool.max_active", 100)
	v.SetDefault("redis.pool.idle_timeout", "5m")
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")

	// Service discovery defaults
	v.SetDefault("service.user.url", "http://user-service:8080")
	v.SetDefault("service.strategy.url", "http://strategy-service:8080")
	v.SetDefault("service.historical.url", "http://historical-service:8080")
}

// Duration is a wrapper around time.Duration to implement yaml.Unmarshaler
type Duration struct {
	time.Duration
}

// UnmarshalText implements encoding.TextUnmarshaler
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
