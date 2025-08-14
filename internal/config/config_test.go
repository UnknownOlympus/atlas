package config_test

import (
	"testing"
	"time"

	"github.com/UnknownOlympus/atlas/internal/config"
	"github.com/stretchr/testify/assert"
)

func Test_MustLoadFromFile(t *testing.T) {
	t.Setenv("ATLAS_ENV", "local")
	t.Setenv("ATLAS_INTERVAL", "10m")
	t.Setenv("ATLAS_PROVIDER_API_KEY", "testAPIKey")
	t.Setenv("DB_HOST", "testHost")
	t.Setenv("DB_PORT", "12345")
	t.Setenv("DB_USERNAME", "admin")
	t.Setenv("DB_PASSWORD", "adminpass")
	t.Setenv("DB_NAME", "testName")

	cfg := config.MustLoad()

	assert.Equal(t, "local", cfg.Env)
	assert.Equal(t, "testHost", cfg.Database.Host)
	assert.Equal(t, "12345", cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.User)
	assert.Equal(t, "adminpass", cfg.Database.Password)
	assert.Equal(t, "testName", cfg.Database.Name)
	assert.Equal(t, 10*time.Minute, cfg.Interval)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "testAPIKey", cfg.APIKey)
	assert.Equal(t, 10, cfg.Workers)
}

func TestMustLoad_IntervalError(t *testing.T) {
	t.Setenv("ATLAS_INTERVAL", "error_value")

	assert.PanicsWithValue(t, "failed to parse interval from configuration", func() {
		config.MustLoad()
	})
}

func TestMustLoad_PortError(t *testing.T) {
	t.Setenv("ATLAS_HEALTH_PORT", "error_value")

	assert.PanicsWithValue(t, "failed to parse port for monitoring server from configuration", func() {
		config.MustLoad()
	})
}

func TestMustLoad_WorkersError(t *testing.T) {
	t.Setenv("ATLAS_WORKERS", "error_value")

	assert.PanicsWithValue(t, "failed to parse workers from configuration, must be an integer types", func() {
		config.MustLoad()
	})
}
