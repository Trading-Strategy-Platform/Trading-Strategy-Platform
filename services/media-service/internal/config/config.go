package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server  ServerConfig
	Storage StorageConfig
	Auth    AuthConfig
	Logging LoggingConfig
	Upload  UploadConfig
}

// ServerConfig holds server specific configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type  string // "local", "s3", "gcs"
	Local LocalStorageConfig
	S3    S3StorageConfig
}

// LocalStorageConfig holds local storage configuration
type LocalStorageConfig struct {
	BasePath    string
	BaseURL     string
	Permissions string
}

// S3StorageConfig holds AWS S3 configuration
type S3StorageConfig struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	BaseURL   string
}

// AuthConfig holds authentication specific configuration
type AuthConfig struct {
	Enabled    bool
	ServiceKey string
}

// UploadConfig holds upload configuration
type UploadConfig struct {
	MaxFileSize       int64
	AllowedExtensions []string
	MaxWidth          int
	MaxHeight         int
	ThumbnailSizes    []ThumbnailSize
}

// ThumbnailSize defines a thumbnail size
type ThumbnailSize struct {
	Name   string
	Width  int
	Height int
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

	// Storage defaults
	v.SetDefault("storage.type", "local")
	v.SetDefault("storage.local.basePath", "/data/images")
	v.SetDefault("storage.local.baseURL", "http://localhost:8080/api/v1/media")
	v.SetDefault("storage.local.permissions", "0644")

	// Auth defaults
	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.serviceKey", "media-service-key")

	// Upload defaults
	v.SetDefault("upload.maxFileSize", 10485760) // 10MB
	v.SetDefault("upload.allowedExtensions", []string{".jpg", ".jpeg", ".png", ".gif"})
	v.SetDefault("upload.maxWidth", 4096)
	v.SetDefault("upload.maxHeight", 4096)
	v.SetDefault("upload.thumbnailSizes", []map[string]interface{}{
		{"name": "small", "width": 150, "height": 150},
		{"name": "medium", "width": 300, "height": 300},
		{"name": "large", "width": 600, "height": 600},
	})

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}
