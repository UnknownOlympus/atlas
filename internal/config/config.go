package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the configuration settings for the geocoding service.
// It includes the environment, server port, API key, number of workers,
// interval for processing, and database configuration.
//
// Fields:
// - Env: The current environment (e.g., local, dev, prod).
// - Port: The port for the geocoder monitoring server.
// - ProviderType: The type of geocoding provider to use (google, nominatim).
// - APIKey: The API key for accessing external services (required for Google).
// - Workers: The number of concurrent workers for processing requests.
// - Interval: The duration between processing intervals.
// - Database: Configuration settings for the PostgreSQL database.
type Config struct {
	Env          string         `yaml:"env"`               // Env is the current environment: local, dev, prod.
	Port         int            `yaml:"geocoder.port"`     // Port is the geocoder monitoring server port.
	ProviderType string         `yaml:"provider.type"`     // ProviderType specifies which geocoding provider to use
	APIKey       string         `yaml:"geocoder.api_key"`  // The API key for accessing external services.
	Workers      int            `yaml:"geocoder.workers"`  // The number of concurrent workers for processing requests.
	Interval     time.Duration  `yaml:"geocoder.interval"` // The duration between processing intervals.
	Database     PostgresConfig `yaml:"postgres"`          // Database holds the postgres database configuration
	AddrPrefix   string         `yaml:"addr_prefix"`       // Address prefix for more accurate geocoding
}

// PostgresConfig struct holds the configuration details for connecting to a PostgreSQL database.
type PostgresConfig struct {
	Host     string `yaml:"host"`                        // Host is the database server address.
	Port     string `yaml:"port"     env-default:"5432"` // Port is the database server port.
	User     string `yaml:"user"`                        // User is the database user.
	Password string `yaml:"password"`                    // Password is the database user's password.
	Name     string `yaml:"db_name"`                     // Name is the name of the database.
}

// MustLoad loads the configuration from a YAML file and returns a Config struct.
func MustLoad() *Config {
	_ = godotenv.Load()

	interval, err := time.ParseDuration(setDeafultEnv("ATLAS_INTERVAL", "10m"))
	if err != nil {
		panic("failed to parse interval from configuration")
	}

	healthPort, err := strconv.Atoi(setDeafultEnv("ATLAS_HEALTH_PORT", "8080"))
	if err != nil {
		panic("failed to parse port for monitoring server from configuration")
	}

	workers, err := strconv.Atoi(setDeafultEnv("ATLAS_WORKERS", "10"))
	if err != nil {
		panic("failed to parse workers from configuration, must be an integer types")
	}

	return &Config{
		Env:          setDeafultEnv("ATLAS_ENV", "production"),
		AddrPrefix:   setDeafultEnv("ATLAS_ADDRESS_PREFIX", ""),
		Port:         healthPort,
		ProviderType: setDeafultEnv("ATLAS_PROVIDER_TYPE", "google"), // Default to Google for backward compatibility
		APIKey:       os.Getenv("ATLAS_PROVIDER_KEY"),
		Workers:      workers,
		Interval:     interval,
		Database: PostgresConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USERNAME"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
		},
	}
}

func setDeafultEnv(key, override string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = override
	}

	return value
}
