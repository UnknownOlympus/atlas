package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the configuration settings for the application.
// It includes the environment type, database configuration,
// token for authentication.
type Config struct {
	Env      string `yaml:"env"`           // Env is the current environment: local, dev, prod.
	Port     int    `yaml:"geocoder.port"` // Port is the geocoder monitoring server port.
	ApiKey   string
	Workers  int
	Interval time.Duration
	Database PostgresConfig `yaml:"postgres"` // Database holds the postgres database configuration
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
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		panic("config path is empty")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		panic("config error: " + err.Error())
	}

	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("geocoder.port", "8080")
	viper.SetDefault("geocoder.workers", "10")
	viper.SetDefault("geocoder.interval", "10m")
	viper.SetDefault("env", "local")

	return &Config{
		Env:      viper.GetString("env"),
		Port:     viper.GetInt("geocoder.port"),
		ApiKey:   viper.GetString("geocoder.api_key"),
		Workers:  viper.GetInt("geocoder.workers"),
		Interval: viper.GetDuration("geocoder.interval"),
		Database: PostgresConfig{
			Host:     viper.GetString("postgres.host"),
			Port:     viper.GetString("postgres.port"),
			User:     viper.GetString("postgres.user"),
			Password: viper.GetString("postgres.password"),
			Name:     viper.GetString("postgres.db_name"),
		},
	}
}
